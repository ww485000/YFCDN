#!/usr/bin/env bash
set -euo pipefail

CONTROL_URL="${YFCDN_CONTROL_URL:-}"
NODE_KEY="${YFCDN_NODE_KEY:-}"
NGINX_CONF="${YFCDN_NGINX_CONF:-/etc/nginx/conf.d/yfcdn-managed.conf}"
RELOAD_CMD="${YFCDN_RELOAD_CMD:-nginx -t && systemctl reload nginx}"
SAME_SERVER=0

usage() {
  cat <<'EOF_USAGE'
YFCDN edge node one-command installer.

Usage:
  sudo bash install-node.sh --control http://CONTROL:8080 --node-key node_xxx
  curl -fsSL http://CONTROL:8080/install-node.sh | sudo bash -s -- --control http://CONTROL:8080 --node-key node_xxx
  curl -fsSL http://127.0.0.1:8080/install-node.sh | sudo bash -s -- --control http://127.0.0.1:8080 --node-key node_xxx --same-server

Options:
  --control URL        Control plane URL. Example: http://1.2.3.4:8080
  --node-key KEY      Node key generated in YFCDN panel.
  --same-server       Control plane and edge node are on the same server.
  --nginx-conf PATH   Nginx managed config path. Default: /etc/nginx/conf.d/yfcdn-managed.conf
  --reload-cmd CMD    Reload command. Default: nginx -t && systemctl reload nginx
  -h, --help          Show help.
EOF_USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --control)
      CONTROL_URL="${2:-}"
      shift 2
      ;;
    --node-key)
      NODE_KEY="${2:-}"
      shift 2
      ;;
    --same-server)
      SAME_SERVER=1
      shift
      ;;
    --nginx-conf)
      NGINX_CONF="${2:-}"
      shift 2
      ;;
    --reload-cmd)
      RELOAD_CMD="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [ "${EUID:-$(id -u)}" -ne 0 ]; then
  echo "Please run as root or with sudo." >&2
  exit 1
fi

if [ "$SAME_SERVER" = "1" ] && [ -z "$CONTROL_URL" ]; then
  CONTROL_URL="http://127.0.0.1:8080"
fi

CONTROL_URL="${CONTROL_URL%/}"

if [ -z "$CONTROL_URL" ]; then
  echo "--control is required." >&2
  exit 1
fi

if [ -z "$NODE_KEY" ]; then
  echo "--node-key is required." >&2
  exit 1
fi

log() {
  printf '[YFCDN] %s\n' "$*"
}

install_packages() {
  if command -v apt-get >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y ca-certificates curl nginx
    if ! command -v go >/dev/null 2>&1; then
      apt-get install -y golang-go
    fi
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y ca-certificates curl nginx
    if ! command -v go >/dev/null 2>&1; then
      dnf install -y golang
    fi
  elif command -v yum >/dev/null 2>&1; then
    yum install -y ca-certificates curl nginx
    if ! command -v go >/dev/null 2>&1; then
      yum install -y golang
    fi
  else
    echo "Unsupported Linux distribution: apt-get, dnf or yum is required." >&2
    exit 1
  fi
}

if ! command -v nginx >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1 || ! command -v go >/dev/null 2>&1; then
  log "Installing dependencies: Nginx, curl, Go"
  install_packages
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go compiler is required but was not installed successfully." >&2
  exit 1
fi

log "Building yfcdn-agent"
BUILD_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$BUILD_DIR"
}
trap cleanup EXIT

cat > "$BUILD_DIR/main.go" <<'EOF_AGENT_GO'
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const agentVersion = "0.1.2"

type Envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

type AgentConfig struct {
	GeneratedAt string   `json:"generated_at"`
	Node        NodeInfo `json:"node"`
	Domains     []Domain `json:"domains"`
	NginxConfig string   `json:"nginx_config"`
}

type NodeInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Region string `json:"region"`
	ISP    string `json:"isp"`
	Status string `json:"status"`
}

type Domain struct {
	ID        string   `json:"id"`
	Hostname  string   `json:"hostname"`
	Origin    string   `json:"origin"`
	CacheTTL  int      `json:"cache_ttl"`
	HTTPSMode string   `json:"https_mode"`
	WAFMode   string   `json:"waf_mode"`
	Status    string   `json:"status"`
	NodeIDs   []string `json:"node_ids"`
}

type Metrics struct {
	CPUPercent  float64 `json:"cpu_percent"`
	Load1       float64 `json:"load1"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
	RxBytes     uint64  `json:"rx_bytes"`
	TxBytes     uint64  `json:"tx_bytes"`
}

type CPUStat struct {
	Idle  uint64
	Total uint64
}

func main() {
	api := flag.String("api", getenv("YFCDN_API", "http://127.0.0.1:8080"), "control plane base URL")
	nodeKey := flag.String("node-key", getenv("YFCDN_NODE_KEY", ""), "node key from control plane")
	confPath := flag.String("nginx-conf", getenv("YFCDN_NGINX_CONF", "./yfcdn-managed.conf"), "managed nginx config output path")
	reloadCmd := flag.String("reload-cmd", getenv("YFCDN_RELOAD_CMD", ""), "command to reload nginx after config changes")
	interval := flag.Duration("interval", 20*time.Second, "sync interval")
	dryRun := flag.Bool("dry-run", false, "print config and do not write or reload")
	flag.Parse()

	if strings.TrimSpace(*nodeKey) == "" {
		log.Fatal("missing -node-key or YFCDN_NODE_KEY")
	}
	baseURL := strings.TrimRight(*api, "/")
	client := &http.Client{Timeout: 15 * time.Second}
	log.Printf("YFCDN agent %s started, api=%s", agentVersion, baseURL)

	prevCPU := readCPUStat()
	for {
		metrics := collectMetrics(prevCPU)
		prevCPU = readCPUStat()
		if err := sendHeartbeat(client, baseURL, *nodeKey, metrics); err != nil {
			log.Printf("heartbeat error: %v", err)
		}
		cfg, err := fetchConfig(client, baseURL, *nodeKey)
		if err != nil {
			log.Printf("config error: %v", err)
		} else {
			changed, err := applyConfig(*confPath, cfg.NginxConfig, *dryRun)
			if err != nil {
				log.Printf("apply config error: %v", err)
			} else if changed {
				log.Printf("config updated: %s domains=%d", *confPath, len(cfg.Domains))
				if *reloadCmd != "" && !*dryRun {
					if err := runReload(*reloadCmd); err != nil {
						log.Printf("reload error: %v", err)
					}
				}
			} else {
				log.Printf("config unchanged: domains=%d", len(cfg.Domains))
			}
		}
		time.Sleep(*interval)
	}
}

func sendHeartbeat(client *http.Client, baseURL, nodeKey string, metrics Metrics) error {
	body, _ := json.Marshal(metrics)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Key", nodeKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func fetchConfig(client *http.Client, baseURL, nodeKey string) (*AgentConfig, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/agent/config", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Node-Key", nodeKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error == "" {
			env.Error = resp.Status
		}
		return nil, errors.New(env.Error)
	}
	var cfg AgentConfig
	if err := json.Unmarshal(env.Data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyConfig(path, content string, dryRun bool) (bool, error) {
	if content == "" {
		content = "# Managed by YFCDN\n"
	}
	old, _ := os.ReadFile(path)
	if string(old) == content {
		return false, nil
	}
	if dryRun {
		fmt.Println(content)
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, path)
}

func runReload(cmd string) error {
	c := exec.Command("/bin/sh", "-c", cmd)
	out, err := c.CombinedOutput()
	if len(out) > 0 {
		log.Printf("reload output: %s", strings.TrimSpace(string(out)))
	}
	return err
}

func collectMetrics(prev CPUStat) Metrics {
	load1 := readLoad1()
	memUsed, memTotal := readMem()
	rx, tx := readNetDev()
	cpu := 0.0
	cur := readCPUStat()
	if prev.Total > 0 && cur.Total > prev.Total {
		totalDelta := cur.Total - prev.Total
		idleDelta := cur.Idle - prev.Idle
		if totalDelta > 0 && idleDelta <= totalDelta {
			cpu = (1 - float64(idleDelta)/float64(totalDelta)) * 100
		}
	}
	return Metrics{CPUPercent: round2(cpu), Load1: load1, MemoryUsed: memUsed, MemoryTotal: memTotal, RxBytes: rx, TxBytes: tx}
}

func readLoad1() float64 {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return v
}

func readMem() (used uint64, total uint64) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var memTotal, memAvailable uint64
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			memTotal = v * 1024
		case "MemAvailable":
			memAvailable = v * 1024
		}
	}
	if memTotal > memAvailable {
		return memTotal - memAvailable, memTotal
	}
	return 0, memTotal
}

func readNetDev() (rx uint64, tx uint64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rxv, _ := strconv.ParseUint(fields[0], 10, 64)
		txv, _ := strconv.ParseUint(fields[8], 10, 64)
		rx += rxv
		tx += txv
	}
	return rx, tx
}

func readCPUStat() CPUStat {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return CPUStat{}
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		var nums []uint64
		for _, f := range fields[1:] {
			v, _ := strconv.ParseUint(f, 10, 64)
			nums = append(nums, v)
		}
		var total uint64
		for _, v := range nums {
			total += v
		}
		idle := nums[3]
		if len(nums) > 4 {
			idle += nums[4]
		}
		return CPUStat{Idle: idle, Total: total}
	}
	return CPUStat{}
}

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

EOF_AGENT_GO

(
  cd "$BUILD_DIR"
  GO111MODULE=off CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /usr/local/bin/yfcdn-agent main.go
)

chmod 0755 /usr/local/bin/yfcdn-agent

log "Writing configuration"
mkdir -p /etc/yfcdn /var/cache/yfcdn/tmp /var/log/nginx "$(dirname "$NGINX_CONF")"

{
  printf 'YFCDN_API=%q\n' "$CONTROL_URL"
  printf 'YFCDN_NODE_KEY=%q\n' "$NODE_KEY"
  printf 'YFCDN_NGINX_CONF=%q\n' "$NGINX_CONF"
  printf 'YFCDN_RELOAD_CMD=%q\n' "$RELOAD_CMD"
} > /etc/yfcdn/agent.env
chmod 0600 /etc/yfcdn/agent.env

cat > /usr/local/bin/yfcdn-agent-start <<'EOF_START'
#!/usr/bin/env bash
set -euo pipefail
source /etc/yfcdn/agent.env
exec /usr/local/bin/yfcdn-agent \
  -api "$YFCDN_API" \
  -node-key "$YFCDN_NODE_KEY" \
  -nginx-conf "$YFCDN_NGINX_CONF" \
  -reload-cmd "$YFCDN_RELOAD_CMD"
EOF_START
chmod 0755 /usr/local/bin/yfcdn-agent-start

if command -v nginx >/dev/null 2>&1; then
  nginx -t
fi

if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
  cat > /etc/systemd/system/yfcdn-agent.service <<'EOF_SERVICE'
[Unit]
Description=YFCDN Edge Agent
After=network-online.target nginx.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/yfcdn-agent-start
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF_SERVICE

  systemctl daemon-reload
  systemctl enable --now nginx || true
  systemctl restart yfcdn-agent
  systemctl enable yfcdn-agent
  log "Service installed: yfcdn-agent"
  systemctl --no-pager --full status yfcdn-agent || true
else
  log "systemd not detected, starting agent with nohup"
  pkill -f '/usr/local/bin/yfcdn-agent' >/dev/null 2>&1 || true
  nohup /usr/local/bin/yfcdn-agent-start >/var/log/yfcdn-agent.log 2>&1 &
fi

log "Checking control plane health: $CONTROL_URL/api/v1/health"
curl -fsS "$CONTROL_URL/api/v1/health" >/dev/null || log "Warning: control plane health check failed. Check firewall, port 8080 or control URL."

log "Node installation completed."
log "Control URL: $CONTROL_URL"
log "Nginx config: $NGINX_CONF"
log "Same-server mode: $SAME_SERVER"
