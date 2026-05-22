package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "0.1.3"

type contextKey string

const authContextKey contextKey = "auth"

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Salt         string `json:"-"`
	Role         string `json:"role"`
	TenantID     string `json:"tenant_id,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type Session struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

type Tenant struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Contact        string `json:"contact"`
	Status         string `json:"status"`
	TrafficQuotaGB int64  `json:"traffic_quota_gb"`
	Remark         string `json:"remark"`
	CreatedAt      string `json:"created_at"`
}

type Operator struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Contact    string   `json:"contact"`
	Status     string   `json:"status"`
	APIEnabled bool     `json:"api_enabled"`
	TenantIDs  []string `json:"tenant_ids"`
	CreatedAt  string   `json:"created_at"`
}

type Node struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	IP          string  `json:"ip"`
	Region      string  `json:"region"`
	ISP         string  `json:"isp"`
	Status      string  `json:"status"`
	NodeKey     string  `json:"node_key,omitempty"`
	LastSeen    string  `json:"last_seen,omitempty"`
	CPUPercent  float64 `json:"cpu_percent"`
	Load1       float64 `json:"load1"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryTotal uint64  `json:"memory_total"`
	RxBytes     uint64  `json:"rx_bytes"`
	TxBytes     uint64  `json:"tx_bytes"`
	Remark      string  `json:"remark"`
	CreatedAt   string  `json:"created_at"`
}

type Domain struct {
	ID              string   `json:"id"`
	Hostname        string   `json:"hostname"`
	TenantID        string   `json:"tenant_id"`
	Origin          string   `json:"origin"`
	Protocol        string   `json:"protocol"`
	CacheTTL        int      `json:"cache_ttl"`
	HTTPSMode       string   `json:"https_mode"`
	WAFMode         string   `json:"waf_mode"`
	Status          string   `json:"status"`
	NodeIDs         []string `json:"node_ids"`
	VerificationTXT string   `json:"verification_txt"`
	Verified        bool     `json:"verified"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type APIKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	KeyHash   string `json:"-"`
	TenantID  string `json:"tenant_id,omitempty"`
	Scope     string `json:"scope"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type Plan struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	BandwidthGB       int64    `json:"bandwidth_gb"`
	PriceMonthlyCents int64    `json:"price_monthly_cents"`
	Features          []string `json:"features"`
	Status            string   `json:"status"`
	CreatedAt         string   `json:"created_at"`
}

type Event struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Actor     string `json:"actor"`
	CreatedAt string `json:"created_at"`
}

type SystemSettings struct {
	BrandName         string `json:"brand_name"`
	PublicAPIEnabled  bool   `json:"public_api_enabled"`
	DomainVerifyMode  string `json:"domain_verify_mode"`
	AbuseContactEmail string `json:"abuse_contact_email"`
}

type Data struct {
	Users     []User         `json:"users"`
	Sessions  []Session      `json:"sessions"`
	Tenants   []Tenant       `json:"tenants"`
	Operators []Operator     `json:"operators"`
	Nodes     []Node         `json:"nodes"`
	Domains   []Domain       `json:"domains"`
	APIKeys   []APIKey       `json:"api_keys"`
	Plans     []Plan         `json:"plans"`
	Events    []Event        `json:"events"`
	System    SystemSettings `json:"system"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	Data Data `json:"data"`
}

type AuthContext struct {
	Kind   string
	User   *User
	APIKey *APIKey
	Token  string
}

type App struct {
	store     *Store
	staticDir string
}

type responseEnvelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

var hostnameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,251}[a-z0-9]$`)

func main() {
	addr := flag.String("addr", getenv("YFCDN_ADDR", ":8080"), "listen address")
	dataPath := flag.String("data", getenv("YFCDN_DATA", "data/yfcdn.json"), "data file path")
	staticDir := flag.String("static", getenv("YFCDN_STATIC", "web"), "static web directory")
	flag.Parse()

	store, err := LoadStore(*dataPath)
	if err != nil {
		log.Fatalf("load store: %v", err)
	}

	app := &App{store: store, staticDir: *staticDir}
	server := &http.Server{
		Addr:              *addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("YFCDN control plane %s listening on %s", version, *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (app *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", app.handleHealth)
	mux.HandleFunc("/api/v1/auth/login", app.handleLogin)
	mux.HandleFunc("/api/v1/auth/logout", app.withAuth(app.handleLogout))
	mux.HandleFunc("/api/v1/me", app.withAuth(app.handleMe))
	mux.HandleFunc("/api/v1/dashboard", app.withAuth(app.handleDashboard))
	mux.HandleFunc("/api/v1/tenants", app.withAuth(app.handleTenants))
	mux.HandleFunc("/api/v1/tenants/", app.withAuth(app.handleTenantItem))
	mux.HandleFunc("/api/v1/operators", app.withAuth(app.handleOperators))
	mux.HandleFunc("/api/v1/operators/", app.withAuth(app.handleOperatorItem))
	mux.HandleFunc("/api/v1/nodes", app.withAuth(app.handleNodes))
	mux.HandleFunc("/api/v1/nodes/", app.withAuth(app.handleNodeItem))
	mux.HandleFunc("/api/v1/domains", app.withAuth(app.handleDomains))
	mux.HandleFunc("/api/v1/domains/", app.withAuth(app.handleDomainItem))
	mux.HandleFunc("/api/v1/plans", app.withAuth(app.handlePlans))
	mux.HandleFunc("/api/v1/plans/", app.withAuth(app.handlePlanItem))
	mux.HandleFunc("/api/v1/api-keys", app.withAuth(app.handleAPIKeys))
	mux.HandleFunc("/api/v1/api-keys/", app.withAuth(app.handleAPIKeyItem))
	mux.HandleFunc("/api/v1/agent/heartbeat", app.handleAgentHeartbeat)
	mux.HandleFunc("/api/v1/agent/config", app.handleAgentConfig)
	mux.HandleFunc("/", app.handleWeb)
	return withCORS(withLogging(mux))
}

func LoadStore(path string) (*Store, error) {
	store := &Store{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		adminPassword := getenv("YFCDN_ADMIN_PASSWORD", "admin123")
		salt := randomHex(16)
		now := nowString()
		store.Data = Data{
			Users:  []User{{ID: "usr_admin", Username: "admin", PasswordHash: hashPassword(salt, adminPassword), Salt: salt, Role: "admin", CreatedAt: now}},
			Plans:  []Plan{{ID: newID("plan"), Name: "Starter", BandwidthGB: 100, PriceMonthlyCents: 9900, Features: []string{"HTTP cache", "Basic WAF", "API access"}, Status: "active", CreatedAt: now}},
			System: SystemSettings{BrandName: "YFCDN", PublicAPIEnabled: true, DomainVerifyMode: "txt", AbuseContactEmail: "abuse@example.com"},
		}
		store.addEventLocked("system", "Initial admin user created", "system")
		return store, store.saveLocked()
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &store.Data); err != nil {
		return nil, err
	}
	store.migrateLocked()
	return store, store.saveLocked()
}

func (s *Store) migrateLocked() {
	if s.Data.System.BrandName == "" {
		s.Data.System.BrandName = "YFCDN"
	}
	if s.Data.System.DomainVerifyMode == "" {
		s.Data.System.DomainVerifyMode = "txt"
	}
	for i := range s.Data.Nodes {
		if s.Data.Nodes[i].NodeKey == "" {
			s.Data.Nodes[i].NodeKey = "node_" + randomHex(24)
		}
	}
	for i := range s.Data.Domains {
		if s.Data.Domains[i].CacheTTL == 0 {
			s.Data.Domains[i].CacheTTL = 300
		}
		if s.Data.Domains[i].VerificationTXT == "" {
			s.Data.Domains[i].VerificationTXT = "yfcdn-verify=" + randomHex(12)
		}
	}
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) addEventLocked(t, msg, actor string) {
	s.Data.Events = append(s.Data.Events, Event{ID: newID("evt"), Type: t, Message: msg, Actor: actor, CreatedAt: nowString()})
	if len(s.Data.Events) > 200 {
		s.Data.Events = s.Data.Events[len(s.Data.Events)-200:]
	}
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "version": version, "time": nowString()})
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findUserIndex(app.store.Data.Users, req.Username)
	if idx < 0 || app.store.Data.Users[idx].PasswordHash != hashPassword(app.store.Data.Users[idx].Salt, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	token := "sess_" + randomHex(32)
	expiresAt := time.Now().Add(24 * time.Hour).UTC()
	expires := expiresAt.Format(time.RFC3339)
	app.store.Data.Sessions = append(cleanSessions(app.store.Data.Sessions), Session{Token: token, UserID: app.store.Data.Users[idx].ID, ExpiresAt: expires})
	app.store.addEventLocked("auth", "User logged in", app.store.Data.Users[idx].Username)
	if err := app.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	setSessionCookie(w, token, expiresAt)
	writeOK(w, map[string]interface{}{"token": token, "expires_at": expires, "user": app.store.Data.Users[idx]})
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := getAuth(r)
	if ctx == nil || ctx.Token == "" {
		writeOK(w, map[string]string{"status": "ok"})
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	out := make([]Session, 0, len(app.store.Data.Sessions))
	for _, s := range app.store.Data.Sessions {
		if s.Token != ctx.Token {
			out = append(out, s)
		}
	}
	app.store.Data.Sessions = out
	_ = app.store.saveLocked()
	clearSessionCookie(w)
	writeOK(w, map[string]string{"status": "ok"})
}

func (app *App) handleMe(w http.ResponseWriter, r *http.Request) {
	ctx := getAuth(r)
	writeOK(w, map[string]interface{}{"kind": ctx.Kind, "user": ctx.User, "api_key": ctx.APIKey})
}

func (app *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	app.store.mu.RLock()
	defer app.store.mu.RUnlock()
	now := time.Now().UTC()
	onlineNodes := 0
	activeNodes := 0
	activeDomains := 0
	var rx, tx uint64
	for _, n := range app.store.Data.Nodes {
		if n.Status == "online" || n.Status == "active" {
			activeNodes++
		}
		if n.LastSeen != "" {
			if t, err := time.Parse(time.RFC3339, n.LastSeen); err == nil && now.Sub(t) < 2*time.Minute {
				onlineNodes++
			}
		}
		rx += n.RxBytes
		tx += n.TxBytes
	}
	for _, d := range app.store.Data.Domains {
		if d.Status == "active" {
			activeDomains++
		}
	}
	recent := append([]Event(nil), app.store.Data.Events...)
	sort.Slice(recent, func(i, j int) bool { return recent[i].CreatedAt > recent[j].CreatedAt })
	if len(recent) > 10 {
		recent = recent[:10]
	}
	writeOK(w, map[string]interface{}{
		"version":        version,
		"tenants":        len(app.store.Data.Tenants),
		"operators":      len(app.store.Data.Operators),
		"nodes":          len(app.store.Data.Nodes),
		"active_nodes":   activeNodes,
		"online_nodes":   onlineNodes,
		"domains":        len(app.store.Data.Domains),
		"active_domains": activeDomains,
		"rx_bytes":       rx,
		"tx_bytes":       tx,
		"events":         recent,
		"system":         app.store.Data.System,
	})
}

func (app *App) handleTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.store.mu.RLock()
		defer app.store.mu.RUnlock()
		writeOK(w, app.store.Data.Tenants)
	case http.MethodPost:
		var t Tenant
		if err := readJSON(r, &t); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		t.Name = strings.TrimSpace(t.Name)
		if t.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if t.Status == "" {
			t.Status = "active"
		}
		t.ID = newID("ten")
		t.CreatedAt = nowString()
		app.store.mu.Lock()
		defer app.store.mu.Unlock()
		app.store.Data.Tenants = append(app.store.Data.Tenants, t)
		app.store.addEventLocked("tenant", "Tenant created: "+t.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, t)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleTenantItem(w http.ResponseWriter, r *http.Request) {
	id := firstPathPart(strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/"))
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findTenantIndex(app.store.Data.Tenants, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, app.store.Data.Tenants[idx])
	case http.MethodPut:
		var req Tenant
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.ID = app.store.Data.Tenants[idx].ID
		req.CreatedAt = app.store.Data.Tenants[idx].CreatedAt
		if req.Name == "" {
			req.Name = app.store.Data.Tenants[idx].Name
		}
		if req.Status == "" {
			req.Status = app.store.Data.Tenants[idx].Status
		}
		app.store.Data.Tenants[idx] = req
		app.store.addEventLocked("tenant", "Tenant updated: "+req.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, req)
	case http.MethodDelete:
		app.store.Data.Tenants = append(app.store.Data.Tenants[:idx], app.store.Data.Tenants[idx+1:]...)
		app.store.addEventLocked("tenant", "Tenant deleted: "+id, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, map[string]string{"deleted": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleOperators(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.store.mu.RLock()
		defer app.store.mu.RUnlock()
		writeOK(w, app.store.Data.Operators)
	case http.MethodPost:
		var op Operator
		if err := readJSON(r, &op); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		op.Name = strings.TrimSpace(op.Name)
		if op.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if op.Status == "" {
			op.Status = "active"
		}
		op.ID = newID("opr")
		op.CreatedAt = nowString()
		app.store.mu.Lock()
		defer app.store.mu.Unlock()
		app.store.Data.Operators = append(app.store.Data.Operators, op)
		app.store.addEventLocked("operator", "Operator created: "+op.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, op)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleOperatorItem(w http.ResponseWriter, r *http.Request) {
	id := firstPathPart(strings.TrimPrefix(r.URL.Path, "/api/v1/operators/"))
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findOperatorIndex(app.store.Data.Operators, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "operator not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, app.store.Data.Operators[idx])
	case http.MethodPut:
		var req Operator
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.ID = app.store.Data.Operators[idx].ID
		req.CreatedAt = app.store.Data.Operators[idx].CreatedAt
		if req.Name == "" {
			req.Name = app.store.Data.Operators[idx].Name
		}
		if req.Status == "" {
			req.Status = app.store.Data.Operators[idx].Status
		}
		app.store.Data.Operators[idx] = req
		app.store.addEventLocked("operator", "Operator updated: "+req.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, req)
	case http.MethodDelete:
		app.store.Data.Operators = append(app.store.Data.Operators[:idx], app.store.Data.Operators[idx+1:]...)
		app.store.addEventLocked("operator", "Operator deleted: "+id, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, map[string]string{"deleted": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.store.mu.RLock()
		defer app.store.mu.RUnlock()
		writeOK(w, app.store.Data.Nodes)
	case http.MethodPost:
		var n Node
		if err := readJSON(r, &n); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		n.Name = strings.TrimSpace(n.Name)
		if n.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if n.IP != "" && net.ParseIP(n.IP) == nil {
			writeError(w, http.StatusBadRequest, "invalid ip")
			return
		}
		if n.Status == "" {
			n.Status = "active"
		}
		n.ID = newID("nod")
		n.NodeKey = "node_" + randomHex(24)
		n.CreatedAt = nowString()
		app.store.mu.Lock()
		defer app.store.mu.Unlock()
		app.store.Data.Nodes = append(app.store.Data.Nodes, n)
		app.store.addEventLocked("node", "Node created: "+n.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, n)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleNodeItem(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/"), "/")
	parts := splitPath(rest)
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	id := parts[0]
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findNodeIndex(app.store.Data.Nodes, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	if len(parts) > 1 && parts[1] == "rotate-key" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		app.store.Data.Nodes[idx].NodeKey = "node_" + randomHex(24)
		app.store.addEventLocked("node", "Node key rotated: "+app.store.Data.Nodes[idx].Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, app.store.Data.Nodes[idx])
		return
	}
	if len(parts) > 1 && parts[1] == "nginx" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		node := app.store.Data.Nodes[idx]
		domains := domainsForNode(app.store.Data.Domains, node.ID)
		cfg := buildNginxConfig(domains)
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "node": node, "config": cfg})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, app.store.Data.Nodes[idx])
	case http.MethodPut:
		var req Node
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		old := app.store.Data.Nodes[idx]
		if req.Name == "" {
			req.Name = old.Name
		}
		if req.Status == "" {
			req.Status = old.Status
		}
		req.ID = old.ID
		req.NodeKey = old.NodeKey
		req.CreatedAt = old.CreatedAt
		req.LastSeen = old.LastSeen
		req.CPUPercent = old.CPUPercent
		req.Load1 = old.Load1
		req.MemoryUsed = old.MemoryUsed
		req.MemoryTotal = old.MemoryTotal
		req.RxBytes = old.RxBytes
		req.TxBytes = old.TxBytes
		app.store.Data.Nodes[idx] = req
		app.store.addEventLocked("node", "Node updated: "+req.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, req)
	case http.MethodDelete:
		app.store.Data.Nodes = append(app.store.Data.Nodes[:idx], app.store.Data.Nodes[idx+1:]...)
		for i := range app.store.Data.Domains {
			app.store.Data.Domains[i].NodeIDs = removeString(app.store.Data.Domains[i].NodeIDs, id)
		}
		app.store.addEventLocked("node", "Node deleted: "+id, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, map[string]string{"deleted": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleDomains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.store.mu.RLock()
		defer app.store.mu.RUnlock()
		writeOK(w, app.store.Data.Domains)
	case http.MethodPost:
		var d Domain
		if err := readJSON(r, &d); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := normalizeDomain(&d); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		now := nowString()
		d.ID = newID("dom")
		d.VerificationTXT = "yfcdn-verify=" + randomHex(12)
		d.CreatedAt = now
		d.UpdatedAt = now
		app.store.mu.Lock()
		defer app.store.mu.Unlock()
		if findDomainIndex(app.store.Data.Domains, d.Hostname) >= 0 {
			writeError(w, http.StatusConflict, "domain already exists")
			return
		}
		app.store.Data.Domains = append(app.store.Data.Domains, d)
		app.store.addEventLocked("domain", "Domain created: "+d.Hostname, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, d)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleDomainItem(w http.ResponseWriter, r *http.Request) {
	id := firstPathPart(strings.TrimPrefix(r.URL.Path, "/api/v1/domains/"))
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findDomainIDIndex(app.store.Data.Domains, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "domain not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, app.store.Data.Domains[idx])
	case http.MethodPut:
		var req Domain
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		old := app.store.Data.Domains[idx]
		if req.Hostname == "" {
			req.Hostname = old.Hostname
		}
		if req.Origin == "" {
			req.Origin = old.Origin
		}
		if req.Status == "" {
			req.Status = old.Status
		}
		if req.Protocol == "" {
			req.Protocol = old.Protocol
		}
		if req.CacheTTL == 0 {
			req.CacheTTL = old.CacheTTL
		}
		if req.HTTPSMode == "" {
			req.HTTPSMode = old.HTTPSMode
		}
		if req.WAFMode == "" {
			req.WAFMode = old.WAFMode
		}
		if len(req.NodeIDs) == 0 {
			req.NodeIDs = old.NodeIDs
		}
		req.ID = old.ID
		req.VerificationTXT = old.VerificationTXT
		req.Verified = old.Verified
		req.CreatedAt = old.CreatedAt
		req.UpdatedAt = nowString()
		if err := normalizeDomain(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		app.store.Data.Domains[idx] = req
		app.store.addEventLocked("domain", "Domain updated: "+req.Hostname, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, req)
	case http.MethodDelete:
		app.store.Data.Domains = append(app.store.Data.Domains[:idx], app.store.Data.Domains[idx+1:]...)
		app.store.addEventLocked("domain", "Domain deleted: "+id, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, map[string]string{"deleted": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handlePlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.store.mu.RLock()
		defer app.store.mu.RUnlock()
		writeOK(w, app.store.Data.Plans)
	case http.MethodPost:
		var p Plan
		if err := readJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if p.Status == "" {
			p.Status = "active"
		}
		p.ID = newID("pln")
		p.CreatedAt = nowString()
		app.store.mu.Lock()
		defer app.store.mu.Unlock()
		app.store.Data.Plans = append(app.store.Data.Plans, p)
		app.store.addEventLocked("plan", "Plan created: "+p.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, p)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handlePlanItem(w http.ResponseWriter, r *http.Request) {
	id := firstPathPart(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"))
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findPlanIndex(app.store.Data.Plans, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "plan not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, app.store.Data.Plans[idx])
	case http.MethodPut:
		var req Plan
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		old := app.store.Data.Plans[idx]
		if req.Name == "" {
			req.Name = old.Name
		}
		if req.Status == "" {
			req.Status = old.Status
		}
		req.ID = old.ID
		req.CreatedAt = old.CreatedAt
		app.store.Data.Plans[idx] = req
		app.store.addEventLocked("plan", "Plan updated: "+req.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, req)
	case http.MethodDelete:
		app.store.Data.Plans = append(app.store.Data.Plans[:idx], app.store.Data.Plans[idx+1:]...)
		app.store.addEventLocked("plan", "Plan deleted: "+id, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, map[string]string{"deleted": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.store.mu.RLock()
		defer app.store.mu.RUnlock()
		writeOK(w, app.store.Data.APIKeys)
	case http.MethodPost:
		var req APIKey
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		secret := "yfc_" + randomHex(32)
		key := APIKey{ID: newID("key"), Name: name, Prefix: secret[:12], KeyHash: hashAPIKey(secret), TenantID: req.TenantID, Scope: req.Scope, Status: "active", CreatedAt: nowString()}
		if key.Scope == "" {
			key.Scope = "admin"
		}
		app.store.mu.Lock()
		defer app.store.mu.Unlock()
		app.store.Data.APIKeys = append(app.store.Data.APIKeys, key)
		app.store.addEventLocked("api_key", "API key created: "+key.Name, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, map[string]interface{}{"api_key": key, "generated_key": secret})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleAPIKeyItem(w http.ResponseWriter, r *http.Request) {
	id := firstPathPart(strings.TrimPrefix(r.URL.Path, "/api/v1/api-keys/"))
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findAPIKeyIndex(app.store.Data.APIKeys, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, "api key not found")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		app.store.Data.APIKeys[idx].Status = "revoked"
		app.store.addEventLocked("api_key", "API key revoked: "+id, authActor(r))
		if err := app.store.saveLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeOK(w, app.store.Data.APIKeys[idx])
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (app *App) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	nodeKey := nodeKeyFromRequest(r)
	if nodeKey == "" {
		writeError(w, http.StatusUnauthorized, "missing node key")
		return
	}
	var req struct {
		CPUPercent  float64 `json:"cpu_percent"`
		Load1       float64 `json:"load1"`
		MemoryUsed  uint64  `json:"memory_used"`
		MemoryTotal uint64  `json:"memory_total"`
		RxBytes     uint64  `json:"rx_bytes"`
		TxBytes     uint64  `json:"tx_bytes"`
	}
	_ = readJSON(r, &req)
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findNodeKeyIndex(app.store.Data.Nodes, nodeKey)
	if idx < 0 {
		writeError(w, http.StatusUnauthorized, "invalid node key")
		return
	}
	app.store.Data.Nodes[idx].LastSeen = nowString()
	if app.store.Data.Nodes[idx].Status == "active" || app.store.Data.Nodes[idx].Status == "offline" || app.store.Data.Nodes[idx].Status == "" {
		app.store.Data.Nodes[idx].Status = "online"
	}
	app.store.Data.Nodes[idx].CPUPercent = req.CPUPercent
	app.store.Data.Nodes[idx].Load1 = req.Load1
	app.store.Data.Nodes[idx].MemoryUsed = req.MemoryUsed
	app.store.Data.Nodes[idx].MemoryTotal = req.MemoryTotal
	app.store.Data.Nodes[idx].RxBytes = req.RxBytes
	app.store.Data.Nodes[idx].TxBytes = req.TxBytes
	if err := app.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"status": "ok", "node_id": app.store.Data.Nodes[idx].ID})
}

func (app *App) handleAgentConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	nodeKey := nodeKeyFromRequest(r)
	if nodeKey == "" {
		writeError(w, http.StatusUnauthorized, "missing node key")
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	idx := findNodeKeyIndex(app.store.Data.Nodes, nodeKey)
	if idx < 0 {
		writeError(w, http.StatusUnauthorized, "invalid node key")
		return
	}
	app.store.Data.Nodes[idx].LastSeen = nowString()
	domains := domainsForNode(app.store.Data.Domains, app.store.Data.Nodes[idx].ID)
	cfg := map[string]interface{}{
		"generated_at": nowString(),
		"node":         app.store.Data.Nodes[idx],
		"domains":      domains,
		"nginx_config": buildNginxConfig(domains),
	}
	_ = app.store.saveLocked()
	writeOK(w, cfg)
}

func (app *App) handleWeb(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if path == "." || path == "" {
		path = "index.html"
	}
	full := filepath.Join(app.staticDir, path)
	if !strings.HasPrefix(full, filepath.Clean(app.staticDir)) {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(full); err != nil {
		full = filepath.Join(app.staticDir, "index.html")
	}
	if strings.HasSuffix(full, ".html") || strings.HasSuffix(full, ".js") || strings.HasSuffix(full, ".css") {
		w.Header().Set("Cache-Control", "no-store")
	}
	http.ServeFile(w, r, full)
}

func (app *App) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ctx, err := app.authenticate(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), authContextKey, ctx)))
	}
}

func sessionTokenFromRequest(r *http.Request) string {
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.Header.Get("X-YFCDN-Session")); token != "" {
		return token
	}
	if token := strings.TrimSpace(r.Header.Get("X-Auth-Token")); token != "" {
		return token
	}
	if cookie, err := r.Cookie("yfcdn_session"); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token
		}
	}
	if token := strings.TrimSpace(r.URL.Query().Get("access_token")); token != "" {
		return token
	}
	return ""
}

func (app *App) authenticate(r *http.Request) (*AuthContext, error) {
	token := sessionTokenFromRequest(r)
	apiKey := strings.TrimSpace(r.Header.Get("X-YFCDN-API-Key"))
	if apiKey == "" && strings.HasPrefix(token, "yfc_") {
		apiKey = token
		token = ""
	}
	app.store.mu.RLock()
	defer app.store.mu.RUnlock()
	if token != "" {
		for _, sess := range app.store.Data.Sessions {
			if sess.Token != token {
				continue
			}
			exp, err := time.Parse(time.RFC3339, sess.ExpiresAt)
			if err != nil || time.Now().UTC().After(exp) {
				return nil, errors.New("session expired")
			}
			for i := range app.store.Data.Users {
				if app.store.Data.Users[i].ID == sess.UserID {
					user := app.store.Data.Users[i]
					return &AuthContext{Kind: "user", User: &user, Token: token}, nil
				}
			}
		}
	}
	if apiKey != "" {
		h := hashAPIKey(apiKey)
		for i := range app.store.Data.APIKeys {
			if app.store.Data.APIKeys[i].KeyHash == h && app.store.Data.APIKeys[i].Status == "active" {
				key := app.store.Data.APIKeys[i]
				return &AuthContext{Kind: "api_key", APIKey: &key}, nil
			}
		}
	}
	return nil, errors.New("authentication required")
}

func getAuth(r *http.Request) *AuthContext {
	v := r.Context().Value(authContextKey)
	if v == nil {
		return nil
	}
	ctx, _ := v.(*AuthContext)
	return ctx
}

func authActor(r *http.Request) string {
	ctx := getAuth(r)
	if ctx == nil {
		return "system"
	}
	if ctx.User != nil {
		return ctx.User.Username
	}
	if ctx.APIKey != nil {
		return "api:" + ctx.APIKey.Prefix
	}
	return ctx.Kind
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if len(header) >= 7 && strings.EqualFold(header[:7], "Bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "yfcdn_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "yfcdn_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func normalizeDomain(d *Domain) error {
	d.Hostname = strings.ToLower(strings.TrimSpace(d.Hostname))
	if !hostnameRE.MatchString(d.Hostname) || strings.Contains(d.Hostname, "..") {
		return errors.New("invalid hostname")
	}
	d.Origin = strings.TrimSpace(d.Origin)
	if !strings.HasPrefix(d.Origin, "http://") && !strings.HasPrefix(d.Origin, "https://") {
		return errors.New("origin must start with http:// or https://")
	}
	if d.Protocol == "" {
		d.Protocol = "http"
	}
	if d.CacheTTL <= 0 {
		d.CacheTTL = 300
	}
	if d.HTTPSMode == "" {
		d.HTTPSMode = "off"
	}
	if d.WAFMode == "" {
		d.WAFMode = "basic"
	}
	if d.Status == "" {
		d.Status = "pending"
	}
	d.NodeIDs = uniqueStrings(d.NodeIDs)
	return nil
}

func domainsForNode(domains []Domain, nodeID string) []Domain {
	out := []Domain{}
	for _, d := range domains {
		if d.Status != "active" {
			continue
		}
		if len(d.NodeIDs) == 0 || containsString(d.NodeIDs, nodeID) {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out
}

func buildNginxConfig(domains []Domain) string {
	var b strings.Builder
	b.WriteString("# Managed by YFCDN. Do not edit manually.\n")
	b.WriteString("proxy_cache_path /var/cache/yfcdn levels=1:2 keys_zone=yfcdn_cache:512m max_size=10g inactive=60m use_temp_path=off;\n")
	b.WriteString("proxy_temp_path /var/cache/yfcdn/tmp;\n")
	b.WriteString("map $http_upgrade $connection_upgrade { default upgrade; '' close; }\n\n")
	for _, d := range domains {
		host := safeNginxToken(d.Hostname)
		origin := safeOrigin(d.Origin)
		if host == "" || origin == "" {
			continue
		}
		ttl := d.CacheTTL
		if ttl < 1 {
			ttl = 300
		}
		b.WriteString("server {\n")
		b.WriteString("    listen 80;\n")
		b.WriteString("    server_name " + host + ";\n")
		b.WriteString("    access_log /var/log/nginx/yfcdn_" + strings.ReplaceAll(host, ".", "_") + ".access.log;\n")
		b.WriteString("    error_log /var/log/nginx/yfcdn_" + strings.ReplaceAll(host, ".", "_") + ".error.log warn;\n")
		b.WriteString("    location = /__yfcdn_health { return 200 'ok'; add_header Content-Type text/plain; }\n")
		if d.WAFMode == "basic" || d.WAFMode == "strict" {
			b.WriteString("    if ($request_method !~ ^(GET|HEAD|POST|PUT|PATCH|DELETE|OPTIONS)$) { return 405; }\n")
		}
		b.WriteString("    location / {\n")
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Host $host;\n")
		b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection $connection_upgrade;\n")
		b.WriteString("        proxy_cache yfcdn_cache;\n")
		b.WriteString("        proxy_cache_valid 200 301 302 " + strconv.Itoa(ttl) + "s;\n")
		b.WriteString("        proxy_cache_valid 404 60s;\n")
		b.WriteString("        proxy_cache_bypass $http_cache_control;\n")
		b.WriteString("        add_header X-YFCDN-Cache $upstream_cache_status always;\n")
		b.WriteString("        proxy_pass " + origin + ";\n")
		b.WriteString("    }\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

func safeNginxToken(s string) string {
	s = strings.TrimSpace(s)
	if !hostnameRE.MatchString(s) || strings.ContainsAny(s, " ;{}$`\n\r\t") {
		return ""
	}
	return s
}

func safeOrigin(s string) string {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, " ;{}$`\n\r\t") {
		return ""
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return ""
	}
	return s
}

func writeOK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, responseEnvelope{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, responseEnvelope{OK: false, Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-YFCDN-API-Key, X-YFCDN-Session, X-Auth-Token, X-Node-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func cleanSessions(in []Session) []Session {
	now := time.Now().UTC()
	out := make([]Session, 0, len(in))
	for _, s := range in {
		if t, err := time.Parse(time.RFC3339, s.ExpiresAt); err == nil && t.After(now) {
			out = append(out, s)
		}
	}
	return out
}

func nodeKeyFromRequest(r *http.Request) string {
	key := strings.TrimSpace(r.Header.Get("X-Node-Key"))
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("node_key"))
	}
	return key
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339) }

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func newID(prefix string) string {
	return prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 36) + randomHex(3)
}

func hashPassword(salt, password string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func firstPathPart(s string) string {
	parts := splitPath(s)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func splitPath(s string) []string {
	s = strings.Trim(s, "/")
	if s == "" {
		return nil
	}
	return strings.Split(s, "/")
}

func findUserIndex(items []User, username string) int {
	for i := range items {
		if items[i].Username == username {
			return i
		}
	}
	return -1
}

func findTenantIndex(items []Tenant, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func findOperatorIndex(items []Operator, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func findNodeIndex(items []Node, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func findNodeKeyIndex(items []Node, key string) int {
	for i := range items {
		if items[i].NodeKey == key {
			return i
		}
	}
	return -1
}

func findDomainIndex(items []Domain, hostname string) int {
	for i := range items {
		if items[i].Hostname == hostname {
			return i
		}
	}
	return -1
}

func findDomainIDIndex(items []Domain, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func findPlanIndex(items []Plan, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func findAPIKeyIndex(items []APIKey, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func removeString(in []string, val string) []string {
	out := in[:0]
	for _, v := range in {
		if v != val {
			out = append(out, v)
		}
	}
	return out
}

func containsString(in []string, val string) bool {
	for _, v := range in {
		if v == val {
			return true
		}
	}
	return false
}

func (u User) String() string { return fmt.Sprintf("%s(%s)", u.Username, u.Role) }
