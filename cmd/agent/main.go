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
