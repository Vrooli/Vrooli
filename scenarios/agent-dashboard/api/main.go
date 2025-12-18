package main

import (
	"github.com/vrooli/api-core/preflight"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

const (
	defaultAgentTimeout   = 10 * time.Minute
	maxLogBufferLines     = 2000
	orchestrateMaxTargets = 16
)

var (
	codexManager           *codexAgentManager
	apiRateLimiter         = rate.NewLimiter(rate.Every(100*time.Millisecond), 20)
	orchestrateRateLimiter = rate.NewLimiter(rate.Every(2*time.Second), 1)
	identifierPattern      = regexp.MustCompile(`^[a-zA-Z0-9:_-]+$`)
)

type Agent struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	PID           int                    `json:"pid"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	LastSeen      time.Time              `json:"last_seen"`
	Uptime        string                 `json:"uptime"`
	Command       string                 `json:"command"`
	Task          string                 `json:"task,omitempty"`
	Mode          string                 `json:"mode,omitempty"`
	LogPath       string                 `json:"log_path,omitempty"`
	Capabilities  []string               `json:"capabilities,omitempty"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	RadarPosition *RadarPosition         `json:"radar_position,omitempty"`
	ExitCode      *int                   `json:"exit_code,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

type RadarPosition struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	TargetX float64 `json:"target_x"`
	TargetY float64 `json:"target_y"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type AgentsResponse struct {
	Agents    []*Agent  `json:"agents"`
	Total     int       `json:"total"`
	Running   int       `json:"running"`
	Completed int       `json:"completed"`
	Failed    int       `json:"failed"`
	Stopped   int       `json:"stopped"`
	Timestamp time.Time `json:"timestamp"`
}

type ManagerStats struct {
	Total     int
	Running   int
	Completed int
	Failed    int
	Stopped   int
}

type startAgentRequest struct {
	Task           string   `json:"task"`
	Mode           string   `json:"mode,omitempty"`
	Label          string   `json:"label,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type orchestrateRequest struct {
	Mode           string   `json:"mode"`
	Objective      string   `json:"objective"`
	Targets        []string `json:"targets,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type managedAgent struct {
	mu            sync.RWMutex
	agent         *Agent
	cancel        context.CancelFunc
	cmd           *exec.Cmd
	logPath       string
	logFile       *os.File
	logs          []string
	exitCodeValue int
	exitCodeSet   bool
	done          chan struct{}
}

type codexAgentManager struct {
	mu             sync.RWMutex
	agents         map[string]*managedAgent
	logDir         string
	defaultTimeout time.Duration
	scenarioRoot   string
}

func newCodexAgentManager(logDir string, defaultTimeout time.Duration, scenarioRoot string) (*codexAgentManager, error) {
	if defaultTimeout <= 0 {
		defaultTimeout = defaultAgentTimeout
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	return &codexAgentManager{
		agents:         make(map[string]*managedAgent),
		logDir:         logDir,
		defaultTimeout: defaultTimeout,
		scenarioRoot:   scenarioRoot,
	}, nil
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "agent-dashboard",
	}) {
		return // Process was re-exec'd after rebuild
	}

	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("❌ Invalid API_PORT value: %s", port)
	}

	scenarioRoot := detectScenarioRoot()
	vrooliRoot := detectVrooliRoot(scenarioRoot)
	logDir := filepath.Join(vrooliRoot, ".vrooli", "logs", "scenarios", "agent-dashboard")
	defaultTimeout := resolveCodexAgentTimeout(scenarioRoot)

	manager, err := newCodexAgentManager(logDir, defaultTimeout, scenarioRoot)
	if err != nil {
		log.Fatalf("failed to initialize codex agent manager: %v", err)
	}
	codexManager = manager

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	http.HandleFunc("/health", corsMiddleware(healthHandler))
	http.HandleFunc("/api/v1/version", corsMiddleware(rateLimitMiddleware(apiRateLimiter, versionHandler)))
	http.HandleFunc("/api/v1/agents", corsMiddleware(rateLimitMiddleware(apiRateLimiter, agentsHandler)))
	http.HandleFunc("/api/v1/agents/status", corsMiddleware(rateLimitMiddleware(apiRateLimiter, statusHandler)))
	http.HandleFunc("/api/v1/agents/scan", corsMiddleware(rateLimitMiddleware(apiRateLimiter, scanHandler)))
	http.HandleFunc("/api/v1/capabilities", corsMiddleware(rateLimitMiddleware(apiRateLimiter, capabilitiesHandler)))
	http.HandleFunc("/api/v1/agents/search", corsMiddleware(rateLimitMiddleware(apiRateLimiter, searchByCapabilityHandler)))
	http.HandleFunc("/api/v1/agents/", corsMiddleware(rateLimitMiddleware(apiRateLimiter, individualAgentHandler)))
	http.HandleFunc("/api/v1/orchestrate", corsMiddleware(rateLimitMiddleware(orchestrateRateLimiter, orchestrateHandler)))

	log.Printf("Agent Dashboard API (resource-codex integration) listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "agent-dashboard-api",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true,
	}
	json.NewEncoder(w).Encode(health)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	version := map[string]interface{}{
		"service":             "agent-dashboard",
		"api_version":         "1.1.0",
		"codex_default_mode":  "auto",
		"default_timeout_sec": int(codexManager.defaultTimeout.Seconds()),
	}
	json.NewEncoder(w).Encode(version)
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		agents, stats := codexManager.Snapshot()
		response := AgentsResponse{
			Agents:    agents,
			Total:     stats.Total,
			Running:   stats.Running,
			Completed: stats.Completed,
			Failed:    stats.Failed,
			Stopped:   stats.Stopped,
			Timestamp: time.Now().UTC(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	case http.MethodPost:
		var req startAgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		agent, err := codexManager.Start(req)
		if err != nil {
			errorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(w, APIResponse{Success: true, Data: agent}, http.StatusCreated)
	default:
		errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	stats := codexManager.Summary()
	data := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"running":   stats.Running,
		"completed": stats.Completed,
		"failed":    stats.Failed,
		"stopped":   stats.Stopped,
		"total":     stats.Total,
	}
	jsonResponse(w, APIResponse{Success: true, Data: data}, http.StatusOK)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	msg := "Codex agent manager uses live run tracking; manual scans are not required."
	jsonResponse(w, APIResponse{Success: true, Data: map[string]string{"message": msg}}, http.StatusOK)
}

func capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agents, _ := codexManager.Snapshot()
	capabilityMap := make(map[string]int)
	for _, agent := range agents {
		for _, cap := range agent.Capabilities {
			if cap == "" {
				continue
			}
			capabilityMap[strings.ToLower(cap)]++
		}
	}
	type capabilityInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	capabilities := make([]capabilityInfo, 0, len(capabilityMap))
	for name, count := range capabilityMap {
		capabilities = append(capabilities, capabilityInfo{Name: name, Count: count})
	}
	sort.Slice(capabilities, func(i, j int) bool {
		if capabilities[i].Count != capabilities[j].Count {
			return capabilities[i].Count > capabilities[j].Count
		}
		return capabilities[i].Name < capabilities[j].Name
	})
	jsonResponse(w, APIResponse{Success: true, Data: map[string]interface{}{
		"capabilities": capabilities,
		"total":        len(capabilities),
	}}, http.StatusOK)
}

func searchByCapabilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	capability := strings.TrimSpace(r.URL.Query().Get("capability"))
	if capability == "" {
		errorResponse(w, "Missing 'capability' query parameter", http.StatusBadRequest)
		return
	}
	if len(capability) > 100 {
		errorResponse(w, "Capability parameter too long (max 100 characters)", http.StatusBadRequest)
		return
	}
	validCapability := regexp.MustCompile(`^[a-zA-Z0-9\s\-_]+$`)
	if !validCapability.MatchString(capability) {
		errorResponse(w, "Invalid capability parameter", http.StatusBadRequest)
		return
	}

	agents, _ := codexManager.Snapshot()
	lowerCapability := strings.ToLower(capability)
	matches := make([]*Agent, 0)
	for _, agent := range agents {
		for _, cap := range agent.Capabilities {
			if strings.Contains(strings.ToLower(cap), lowerCapability) {
				matches = append(matches, agent)
				break
			}
		}
	}

	jsonResponse(w, APIResponse{Success: true, Data: map[string]interface{}{
		"capability": capability,
		"agents":     matches,
		"count":      len(matches),
	}}, http.StatusOK)
}

func individualAgentHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		errorResponse(w, "Agent ID or name required", http.StatusBadRequest)
		return
	}

	identifier := parts[0]
	if len(identifier) > 100 || !identifierPattern.MatchString(identifier) {
		errorResponse(w, "Invalid agent identifier", http.StatusBadRequest)
		return
	}

	agentID := resolveAgentIdentifier(identifier)
	if agentID == "" {
		errorResponse(w, "Agent not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if len(parts) == 1 {
			getAgentDetails(w, r, agentID)
			return
		}
		switch parts[1] {
		case "logs":
			getAgentLogs(w, r, agentID)
		case "metrics":
			getAgentMetrics(w, r, agentID)
		default:
			errorResponse(w, "Unknown endpoint", http.StatusNotFound)
		}
	case http.MethodPost:
		if len(parts) != 2 {
			errorResponse(w, "Invalid path", http.StatusNotFound)
			return
		}
		switch parts[1] {
		case "stop":
			agent, err := codexManager.Stop(agentID)
			if err != nil {
				errorResponse(w, err.Error(), http.StatusBadRequest)
				return
			}
			jsonResponse(w, APIResponse{Success: true, Data: agent}, http.StatusOK)
		case "start":
			errorResponse(w, "Use POST /api/v1/agents to start new Codex runs", http.StatusBadRequest)
		default:
			errorResponse(w, "Unknown endpoint", http.StatusNotFound)
		}
	default:
		errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getAgentDetails(w http.ResponseWriter, r *http.Request, agentID string) {
	agent, err := codexManager.Get(agentID)
	if err != nil {
		errorResponse(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func getAgentLogs(w http.ResponseWriter, r *http.Request, agentID string) {
	linesParam := strings.TrimSpace(r.URL.Query().Get("lines"))
	if linesParam == "" {
		linesParam = "200"
	}
	if !isValidLineCount(linesParam) {
		errorResponse(w, "Invalid line count. Must be between 1 and 10000", http.StatusBadRequest)
		return
	}
	lineCount, _ := strconv.Atoi(linesParam)
	logs, err := codexManager.Logs(agentID, lineCount)
	if err != nil {
		errorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, APIResponse{Success: true, Data: map[string]interface{}{
		"agent_id":  agentID,
		"logs":      logs,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}}, http.StatusOK)
}

func getAgentMetrics(w http.ResponseWriter, r *http.Request, agentID string) {
	metrics, err := codexManager.Metrics(agentID)
	if err != nil {
		errorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func orchestrateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req orchestrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Mode == "" {
		req.Mode = "auto"
	}
	if len(req.Targets) > orchestrateMaxTargets {
		req.Targets = req.Targets[:orchestrateMaxTargets]
	}
	prompt := buildOrchestrationPrompt(req)

	startReq := startAgentRequest{
		Task:           prompt,
		Mode:           req.Mode,
		Label:          fmt.Sprintf("Codex orchestration (%s)", strings.ToLower(req.Mode)),
		TimeoutSeconds: req.TimeoutSeconds,
		Capabilities:   []string{"codex", "orchestration", "coordination"},
		Notes:          req.Notes,
	}

	agent, err := codexManager.Start(startReq)
	if err != nil {
		errorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, APIResponse{Success: true, Data: map[string]interface{}{
		"task_id":   agent.ID,
		"mode":      req.Mode,
		"status":    agent.Status,
		"agent":     agent,
		"message":   "Codex orchestration task started",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}}, http.StatusOK)
}

func (m *codexAgentManager) Start(req startAgentRequest) (*Agent, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, errors.New("task is required")
	}

	if _, err := exec.LookPath("resource-codex"); err != nil {
		return nil, fmt.Errorf("resource-codex CLI not found: %w", err)
	}

	timeout := m.defaultTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	prompt := buildCodexPrompt(req, m.scenarioRoot)
	cmdArgs := []string{"content", "execute", "--context", "text", "--operation", "analyze", prompt}
	cmd := exec.CommandContext(ctx, "resource-codex", cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = m.scenarioRoot

	env := os.Environ()
	if req.Mode != "" {
		env = append(env, "CODEX_CLI_MODE="+req.Mode)
	}
	env = append(env, fmt.Sprintf("CODEX_TIMEOUT=%d", int(timeout.Seconds())))
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("prepare stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("prepare stderr pipe: %w", err)
	}

	id := fmt.Sprintf("codex:%s", uuid.New().String())
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = deriveLabelFromTask(task)
	}

	capabilities := req.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"codex", "ai-intelligence", "code-analysis", "auto-remediation"}
	}

	logPath := filepath.Join(m.logDir, strings.ReplaceAll(id, ":", "_")+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open log file: %w", err)
	}

	now := time.Now().UTC()
	agent := &Agent{
		ID:            id,
		Name:          label,
		Type:          "codex",
		Status:        "starting",
		StartTime:     now,
		LastSeen:      now,
		Uptime:        "0m",
		Command:       "resource-codex " + strings.Join(cmdArgs, " "),
		Task:          task,
		Mode:          strings.ToLower(req.Mode),
		LogPath:       logPath,
		Capabilities:  cloneStringSlice(capabilities),
		Metrics:       getDefaultMetrics(),
		RadarPosition: generateRadarPosition(),
	}

	managed := &managedAgent{
		agent:   agent,
		cancel:  cancel,
		cmd:     cmd,
		logPath: logPath,
		logFile: logFile,
		logs:    make([]string, 0, 64),
		done:    make(chan struct{}),
	}

	m.mu.Lock()
	m.agents[id] = managed
	m.mu.Unlock()

	if err := cmd.Start(); err != nil {
		cancel()
		_ = logFile.Close()
		m.mu.Lock()
		delete(m.agents, id)
		m.mu.Unlock()
		return nil, fmt.Errorf("start codex agent: %w", err)
	}

	managed.mu.Lock()
	agent.Status = "running"
	if cmd.Process != nil {
		agent.PID = cmd.Process.Pid
	}
	managed.mu.Unlock()

	go managed.captureStream("stdout", stdoutPipe)
	go managed.captureStream("stderr", stderrPipe)
	go managed.waitForExit(ctx, timeout)

	return cloneAgent(agent), nil
}

func (m *codexAgentManager) Stop(agentID string) (*Agent, error) {
	m.mu.RLock()
	managed, ok := m.agents[agentID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	managed.mu.RLock()
	status := managed.agent.Status
	managed.mu.RUnlock()

	if status != "running" && status != "starting" {
		return cloneAgent(managed.agent), nil
	}

	managed.cancel()

	select {
	case <-managed.done:
	case <-time.After(10 * time.Second):
		managed.mu.Lock()
		if managed.cmd.Process != nil {
			_ = managed.cmd.Process.Kill()
		}
		managed.mu.Unlock()
		<-managed.done
	}

	return cloneAgent(managed.agent), nil
}

func (m *codexAgentManager) Get(agentID string) (*Agent, error) {
	m.mu.RLock()
	managed, ok := m.agents[agentID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	managed.mu.RLock()
	defer managed.mu.RUnlock()
	return cloneAgent(managed.agent), nil
}

func (m *codexAgentManager) Snapshot() ([]*Agent, ManagerStats) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	stats := ManagerStats{}

	for _, managed := range m.agents {
		managed.mu.RLock()
		agentClone := cloneAgent(managed.agent)
		managed.mu.RUnlock()

		switch agentClone.Status {
		case "running", "starting":
			stats.Running++
			if agentClone.PID > 0 {
				agentClone.Metrics = getProcessMetrics(agentClone.PID)
			}
			agentClone.LastSeen = time.Now().UTC()
			agentClone.Uptime = computeUptime(agentClone.StartTime, nil)
		case "completed":
			stats.Completed++
			agentClone.Uptime = computeUptime(agentClone.StartTime, agentClone.EndTime)
		case "stopped":
			stats.Stopped++
			agentClone.Uptime = computeUptime(agentClone.StartTime, agentClone.EndTime)
		default:
			stats.Failed++
			agentClone.Uptime = computeUptime(agentClone.StartTime, agentClone.EndTime)
		}

		if agentClone.RadarPosition == nil {
			agentClone.RadarPosition = generateRadarPosition()
		}

		agents = append(agents, agentClone)
	}

	stats.Total = len(agents)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].StartTime.After(agents[j].StartTime)
	})

	return agents, stats
}

func (m *codexAgentManager) Summary() ManagerStats {
	_, stats := m.Snapshot()
	return stats
}

func (m *codexAgentManager) Logs(agentID string, maxLines int) ([]string, error) {
	m.mu.RLock()
	managed, ok := m.agents[agentID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	managed.mu.RLock()
	logsCopy := append([]string(nil), managed.logs...)
	managed.mu.RUnlock()

	if maxLines > 0 && len(logsCopy) > maxLines {
		logsCopy = logsCopy[len(logsCopy)-maxLines:]
	}
	return logsCopy, nil
}

func (m *codexAgentManager) Metrics(agentID string) (map[string]interface{}, error) {
	m.mu.RLock()
	managed, ok := m.agents[agentID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	managed.mu.RLock()
	pid := managed.agent.PID
	status := managed.agent.Status
	metrics := cloneMetrics(managed.agent.Metrics)
	managed.mu.RUnlock()

	if status == "running" && pid > 0 {
		return getProcessMetrics(pid), nil
	}
	if metrics == nil {
		metrics = getDefaultMetrics()
	}
	return metrics, nil
}

func (m *managedAgent) captureStream(stream string, reader io.ReadCloser) {
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		m.appendLog(stream, line)
	}
	if err := scanner.Err(); err != nil {
		m.appendLog("system", fmt.Sprintf("stream error (%s): %v", stream, err))
	}
}

func (m *managedAgent) appendLog(stream, line string) {
	entry := fmt.Sprintf("[%s] %s", strings.ToUpper(stream), line)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, entry)
	if len(m.logs) > maxLogBufferLines {
		m.logs = m.logs[len(m.logs)-maxLogBufferLines:]
	}
	if m.logFile != nil {
		fmt.Fprintln(m.logFile, entry)
	}
	m.agent.LastSeen = time.Now().UTC()
}

func (m *managedAgent) waitForExit(ctx context.Context, timeout time.Duration) {
	defer close(m.done)

	err := m.cmd.Wait()
	now := time.Now().UTC()

	m.mu.Lock()
	m.agent.LastSeen = now
	m.agent.EndTime = &now
	if m.logFile != nil {
		_ = m.logFile.Close()
		m.logFile = nil
	}

	status := "completed"
	errorMessage := ""
	exitCode := 0

	if err != nil {
		status = "failed"
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			errorMessage = exitErr.Error()
		} else {
			exitCode = -1
			errorMessage = err.Error()
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status = "timeout"
			errorMessage = fmt.Sprintf("agent timed out after %s", timeout)
		} else if errors.Is(ctx.Err(), context.Canceled) {
			status = "stopped"
			errorMessage = "agent run cancelled"
		}
	} else if m.cmd.ProcessState != nil {
		exitCode = m.cmd.ProcessState.ExitCode()
	}

	m.exitCodeValue = exitCode
	m.exitCodeSet = true
	m.agent.ExitCode = &m.exitCodeValue
	m.agent.Status = status
	m.agent.Error = errorMessage
	if m.agent.Metrics == nil {
		m.agent.Metrics = getDefaultMetrics()
	}
	m.mu.Unlock()
}

func buildCodexPrompt(req startAgentRequest, scenarioRoot string) string {
	builder := strings.Builder{}
	builder.WriteString("# Agent Dashboard Codex Task\n\n")
	builder.WriteString("You are operating inside the Vrooli Agent Dashboard scenario.\n")
	builder.WriteString("Carry out the following task with clear reasoning, note-taking, and actionable outputs.\n\n")
	builder.WriteString("## Task\n")
	builder.WriteString(strings.TrimSpace(req.Task))
	builder.WriteString("\n\n")
	if notes := strings.TrimSpace(req.Notes); notes != "" {
		builder.WriteString("## Additional Context\n")
		builder.WriteString(notes)
		builder.WriteString("\n\n")
	}
	builder.WriteString("## Expectations\n")
	builder.WriteString("- Explain your plan before taking actions.\n")
	builder.WriteString("- Document each action and its result.\n")
	builder.WriteString("- Summarize outcomes and recommended next steps.\n")
	builder.WriteString("- Surface any risks or dependencies that require human follow-up.\n")
	builder.WriteString("\n## Environment\n")
	builder.WriteString("- Scenario Root: " + scenarioRoot + "\n")
	builder.WriteString("- Mode: " + strings.TrimSpace(req.Mode) + "\n")
	builder.WriteString("- Timestamp: " + time.Now().UTC().Format(time.RFC3339) + "\n")
	return builder.String()
}

func buildOrchestrationPrompt(req orchestrateRequest) string {
	builder := strings.Builder{}
	builder.WriteString("# Vrooli Multi-Agent Orchestration\n\n")
	builder.WriteString("Coordinate Codex tooling to evaluate the current agent landscape and produce actionable guidance.\n\n")
	builder.WriteString("## Objective\n")
	if req.Objective != "" {
		builder.WriteString(strings.TrimSpace(req.Objective))
	} else {
		builder.WriteString("Assess active agents, identify gaps, and recommend improvements.")
	}
	builder.WriteString("\n\n## Operating Mode\n")
	builder.WriteString(strings.ToUpper(strings.TrimSpace(req.Mode)))
	builder.WriteString("\n\n")
	if len(req.Targets) > 0 {
		builder.WriteString("## Focus Targets\n")
		for _, target := range req.Targets {
			builder.WriteString("- " + strings.TrimSpace(target) + "\n")
		}
		builder.WriteString("\n")
	}
	if notes := strings.TrimSpace(req.Notes); notes != "" {
		builder.WriteString("## Additional Notes\n")
		builder.WriteString(notes)
		builder.WriteString("\n\n")
	}
	builder.WriteString("## Deliverables\n")
	builder.WriteString("1. Current state assessment of agents and workflows.\n")
	builder.WriteString("2. Risks or blockers that impede progress.\n")
	builder.WriteString("3. Recommended actions for the next orchestration cycle.\n")
	builder.WriteString("4. Metrics or signals that should be tracked going forward.\n")
	return builder.String()
}

func deriveLabelFromTask(task string) string {
	cleaned := strings.TrimSpace(task)
	if cleaned == "" {
		return "Codex Agent"
	}
	if len(cleaned) > 64 {
		cleaned = cleaned[:64]
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "Codex Agent"
	}
	return cleaned
}

func cloneAgent(agent *Agent) *Agent {
	if agent == nil {
		return nil
	}
	clone := *agent
	if agent.Capabilities != nil {
		clone.Capabilities = cloneStringSlice(agent.Capabilities)
	}
	if agent.Metrics != nil {
		clone.Metrics = cloneMetrics(agent.Metrics)
	}
	if agent.RadarPosition != nil {
		rp := *agent.RadarPosition
		clone.RadarPosition = &rp
	}
	if agent.EndTime != nil {
		end := *agent.EndTime
		clone.EndTime = &end
	}
	if agent.ExitCode != nil {
		exit := *agent.ExitCode
		clone.ExitCode = &exit
	}
	return &clone
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}

func cloneMetrics(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	clone := make(map[string]interface{}, len(input))
	for k, v := range input {
		clone[k] = v
	}
	return clone
}

func computeUptime(start time.Time, end *time.Time) string {
	if start.IsZero() {
		return "0m"
	}
	var duration time.Duration
	if end != nil && !end.IsZero() {
		duration = end.Sub(start)
		if duration < 0 {
			duration = 0
		}
	} else {
		duration = time.Since(start)
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func generateRadarPosition() *RadarPosition {
	now := time.Now().UnixNano()
	return &RadarPosition{
		X:       50 + float64(now%80-40),
		Y:       50 + float64((now/3)%80-40),
		TargetX: 50 + float64((now/5)%60-30),
		TargetY: 50 + float64((now/7)%60-30),
	}
}

func getDefaultMetrics() map[string]interface{} {
	return map[string]interface{}{
		"cpu_percent":    nil,
		"memory_mb":      nil,
		"io_read_bytes":  nil,
		"io_write_bytes": nil,
		"thread_count":   nil,
		"fd_count":       nil,
	}
}

func getProcessMetrics(pid int) map[string]interface{} {
	metrics := make(map[string]interface{})
	metrics["cpu_percent"] = getProcessCPU(pid)
	metrics["memory_mb"] = getProcessMemory(pid)
	ioStats := getProcessIO(pid)
	for k, v := range ioStats {
		metrics[k] = v
	}
	metrics["thread_count"] = getProcessThreads(pid)
	metrics["fd_count"] = getProcessFDs(pid)
	return metrics
}

func getProcessCPU(pid int) float64 {
	if pid <= 0 || pid > 4194304 {
		return 0.0
	}
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0.0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0.0
	}
	utime, _ := strconv.ParseFloat(fields[13], 64)
	stime, _ := strconv.ParseFloat(fields[14], 64)
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0.0
	}
	uptimeFields := strings.Fields(string(uptimeData))
	uptime, _ := strconv.ParseFloat(uptimeFields[0], 64)
	totalTime := (utime + stime) / 100
	if uptime > 0 {
		return (totalTime / uptime) * 100
	}
	return 0.0
}

func getProcessMemory(pid int) float64 {
	if pid <= 0 || pid > 4194304 {
		return 0.0
	}
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return 0.0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseFloat(fields[1], 64)
				return kb / 1024.0
			}
		}
	}
	return 0.0
}

func getProcessIO(pid int) map[string]interface{} {
	if pid <= 0 || pid > 4194304 {
		return map[string]interface{}{
			"io_read_bytes":  0,
			"io_write_bytes": 0,
		}
	}
	path := fmt.Sprintf("/proc/%d/io", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]interface{}{
			"io_read_bytes":  0,
			"io_write_bytes": 0,
		}
	}
	stats := map[string]interface{}{
		"io_read_bytes":  int64(0),
		"io_write_bytes": int64(0),
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "read_bytes:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				bytesValue, _ := strconv.ParseInt(fields[1], 10, 64)
				stats["io_read_bytes"] = bytesValue
			}
		}
		if strings.HasPrefix(line, "write_bytes:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				bytesValue, _ := strconv.ParseInt(fields[1], 10, 64)
				stats["io_write_bytes"] = bytesValue
			}
		}
	}
	return stats
}

func getProcessThreads(pid int) int {
	if pid <= 0 || pid > 4194304 {
		return 0
	}
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Threads:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				count, _ := strconv.Atoi(fields[1])
				return count
			}
		}
	}
	return 0
}

func getProcessFDs(pid int) int {
	if pid <= 0 || pid > 4194304 {
		return 0
	}
	fdPath := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdPath)
	if err != nil {
		return 0
	}
	return len(entries)
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func rateLimitMiddleware(limiter *rate.Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limiter != nil && !limiter.Allow() {
			errorResponse(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func errorResponse(w http.ResponseWriter, message string, status int) {
	jsonResponse(w, APIResponse{Success: false, Error: message}, status)
}

func isValidLineCount(value string) bool {
	n, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return n > 0 && n <= 10000
}

// validResourceNames is the list of supported resource types
var validResourceNames = map[string]bool{
	"claude-code":   true,
	"cline":         true,
	"ollama":        true,
	"autogen-studio": true,
	"autogpt":       true,
	"crewai":        true,
	"gemini":        true,
	"langchain":     true,
	"litellm":       true,
	"openrouter":    true,
	"whisper":       true,
	"comfyui":       true,
	"pandas-ai":     true,
	"parlant":       true,
	"huginn":        true,
	"opencode":      true,
	"codex":         true,
}

func isValidResourceName(resource string) bool {
	return validResourceNames[resource]
}

func isValidAgentID(agentID string) bool {
	if agentID == "" {
		return false
	}
	// Agent ID should match pattern: resource:agent-identifier
	// where both parts contain only alphanumeric, colon, hyphen, or underscore
	if !identifierPattern.MatchString(agentID) || !strings.Contains(agentID, ":") {
		return false
	}

	// Ensure both parts before and after the colon are non-empty
	parts := strings.SplitN(agentID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	return true
}

func resolveAgentIdentifier(identifier string) string {
	if codexManager == nil {
		return ""
	}
	codexManager.mu.RLock()
	defer codexManager.mu.RUnlock()
	if _, ok := codexManager.agents[identifier]; ok {
		return identifier
	}
	lower := strings.ToLower(identifier)
	for id, managed := range codexManager.agents {
		managed.mu.RLock()
		name := strings.ToLower(managed.agent.Name)
		typ := strings.ToLower(managed.agent.Type)
		shortID := strings.ToLower(strings.TrimPrefix(id, "codex:"))
		managed.mu.RUnlock()
		if name == lower || typ == lower || shortID == lower {
			return id
		}
	}
	return ""
}

func detectScenarioRoot() string {
	if env := strings.TrimSpace(os.Getenv("SCENARIO_ROOT")); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return filepath.Clean(env)
		}
	}
	if wd, err := os.Getwd(); err == nil {
		dir := filepath.Clean(wd)
		for {
			base := filepath.Base(dir)
			if base == "agent-dashboard" {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "."
}

func detectVrooliRoot(start string) string {
	if env := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); env != "" {
		if info, err := os.Stat(filepath.Join(env, ".vrooli")); err == nil && info.IsDir() {
			return filepath.Clean(env)
		}
	}
	dir := filepath.Clean(start)
	for {
		if info, err := os.Stat(filepath.Join(dir, ".vrooli")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Clean(start)
}

func resolveCodexAgentTimeout(scenarioRoot string) time.Duration {
	configPath := filepath.Join(scenarioRoot, "initialization", "configuration", "codex-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultAgentTimeout
	}
	var cfg struct {
		InvestigationSettings struct {
			TimeoutSeconds int `json:"timeout_seconds"`
		} `json:"investigation_settings"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultAgentTimeout
	}
	if cfg.InvestigationSettings.TimeoutSeconds <= 0 {
		return defaultAgentTimeout
	}
	return time.Duration(cfg.InvestigationSettings.TimeoutSeconds) * time.Second
}
