package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	agentStatusRunning   = "running"
	agentStatusStopping  = "stopping"
	agentStatusStopped   = "stopped"
	agentStatusCompleted = "completed"
	agentStatusFailed    = "failed"
)

var ErrAgentNotFound = errors.New("agent not found")

type AgentInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Label           string            `json:"label,omitempty"`
	Action          string            `json:"action,omitempty"`
	Model           string            `json:"model"`
	Status          string            `json:"status"`
	PromptPreview   string            `json:"prompt_preview,omitempty"`
	PromptLength    int               `json:"prompt_length"`
	StartedAt       time.Time         `json:"started_at"`
	EndedAt         *time.Time        `json:"ended_at,omitempty"`
	DurationSeconds int               `json:"duration_seconds"`
	Error           string            `json:"error,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	PID             int               `json:"pid,omitempty"`
}

type AgentRunRequest struct {
	Prompt   string
	Model    string
	Label    string
	Action   string
	Metadata map[string]string
	Timeout  time.Duration
}

type AgentRunResult struct {
	Output string
	Agent  AgentInfo
}

type AgentManager struct {
	mu           sync.RWMutex
	agents       map[string]*agentInstance
	logs         map[string]string
	logsDir      string
	scenarioRoot string
}

type agentInstance struct {
	mu      sync.Mutex
	info    AgentInfo
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	timer   *time.Timer
	logPath string
	writer  *bufio.Writer
	output  bytes.Buffer
	wg      sync.WaitGroup
}

func NewAgentManager() (*AgentManager, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory: %w", err)
	}

	logsDir := filepath.Join(root, "logs", "agents")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create agent logs directory: %w", err)
	}

	return &AgentManager{
		agents:       make(map[string]*agentInstance),
		logs:         make(map[string]string),
		logsDir:      logsDir,
		scenarioRoot: root,
	}, nil
}

func (am *AgentManager) RunAgent(ctx context.Context, req AgentRunRequest) (*AgentRunResult, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = resolveModel("")
	}

	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "Test Genie Agent"
	}

	action := strings.TrimSpace(req.Action)
	if action == "" {
		action = "generic_task"
	}

	agentID := uuid.New().String()
	logPath := filepath.Join(am.logsDir, fmt.Sprintf("%s.log", agentID))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	promptFile, err := os.CreateTemp(am.logsDir, "prompt-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt file: %w", err)
	}
	if _, err := promptFile.WriteString(prompt); err != nil {
		promptFile.Close()
		os.Remove(promptFile.Name())
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	promptFile.Close()
	defer os.Remove(promptFile.Name())

	cmdCtx := context.Background()
	cancel := func() {}
	if ctx != nil {
		cmdCtx, cancel = context.WithCancel(context.Background())
		go func() {
			select {
			case <-ctx.Done():
				cancel()
			case <-cmdCtx.Done():
			}
		}()
	} else {
		cmdCtx, cancel = context.WithCancel(context.Background())
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	timer := time.AfterFunc(timeout, cancel)

	args := []string{"agents", "run", "--prompt-file", promptFile.Name(), "--model", model}

	cmd := exec.CommandContext(cmdCtx, "resource-opencode", args...)
	cmd.Dir = am.scenarioRoot
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		timer.Stop()
		return nil, fmt.Errorf("failed to attach stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		timer.Stop()
		return nil, fmt.Errorf("failed to attach stderr: %w", err)
	}

	instance := &agentInstance{
		info: AgentInfo{
			ID:            agentID,
			Name:          label,
			Label:         label,
			Action:        action,
			Model:         model,
			Status:        agentStatusRunning,
			PromptPreview: truncatePrompt(prompt, 160),
			PromptLength:  len([]rune(prompt)),
			StartedAt:     time.Now().UTC(),
			Metadata:      cloneMetadata(req.Metadata),
		},
		cmd:     cmd,
		cancel:  cancel,
		timer:   timer,
		logPath: logPath,
	}

	instance.writer = bufio.NewWriter(logFile)
	instance.info.Metadata["log_path"] = logPath

	am.mu.Lock()
	am.agents[agentID] = instance
	am.logs[agentID] = logPath
	am.mu.Unlock()

	if err := cmd.Start(); err != nil {
		timer.Stop()
		am.mu.Lock()
		delete(am.agents, agentID)
		delete(am.logs, agentID)
		am.mu.Unlock()
		return nil, fmt.Errorf("failed to start resource-opencode: %w", err)
	}

	instance.mu.Lock()
	instance.info.PID = cmd.Process.Pid
	instance.mu.Unlock()

	streamCopy := func(reader io.Reader) {
		defer instance.wg.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			instance.appendLine(line)
		}
	}

	instance.wg.Add(2)
	go streamCopy(stdout)
	go streamCopy(stderr)

	err = cmd.Wait()
	timer.Stop()
	instance.wg.Wait()

	instance.finishLogging()

	endTime := time.Now().UTC()

	instance.mu.Lock()
	instance.info.EndedAt = &endTime
	instance.info.DurationSeconds = int(endTime.Sub(instance.info.StartedAt).Seconds())
	switch {
	case err == nil:
		instance.info.Status = agentStatusCompleted
	case errors.Is(err, context.Canceled):
		if instance.info.Status == agentStatusStopping {
			instance.info.Status = agentStatusStopped
		} else {
			instance.info.Status = agentStatusStopped
		}
	default:
		if instance.info.Status == agentStatusStopping {
			instance.info.Status = agentStatusStopped
		} else {
			instance.info.Status = agentStatusFailed
			instance.info.Error = err.Error()
		}
	}
	finalInfo := instance.info
	output := instance.output.String()
	instance.mu.Unlock()

	am.mu.Lock()
	delete(am.agents, agentID)
	am.mu.Unlock()

	return &AgentRunResult{Output: output, Agent: finalInfo}, nil
}

func (am *AgentManager) ListAgents() []AgentInfo {
	am.mu.RLock()
	defer am.mu.RUnlock()

	agents := make([]AgentInfo, 0, len(am.agents))
	for _, inst := range am.agents {
		inst.mu.Lock()
		info := inst.info
		inst.mu.Unlock()
		agents = append(agents, info)
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].StartedAt.Before(agents[j].StartedAt)
	})

	return agents
}

func (am *AgentManager) StopAgent(agentID string) error {
	am.mu.Lock()
	inst, ok := am.agents[agentID]
	if !ok {
		am.mu.Unlock()
		return ErrAgentNotFound
	}
	inst.mu.Lock()
	inst.info.Status = agentStatusStopping
	if inst.timer != nil {
		inst.timer.Stop()
	}
	inst.mu.Unlock()
	cancel := inst.cancel
	cmd := inst.cmd
	am.mu.Unlock()

	cancel()

	if cmd != nil && cmd.Process != nil {
		if runtime.GOOS == "windows" {
			_ = cmd.Process.Kill()
		} else {
			if err := cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
				_ = cmd.Process.Signal(syscall.SIGTERM)
			}
		}
	}

	return nil
}

func (am *AgentManager) AgentLogPath(agentID string) string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.logs[agentID]
}

func (inst *agentInstance) appendLine(line string) {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	if inst.writer != nil {
		inst.writer.WriteString(line)
		inst.writer.WriteByte('\n')
	}
	inst.output.WriteString(line)
	inst.output.WriteByte('\n')
}

func (inst *agentInstance) finishLogging() {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	if inst.writer != nil {
		inst.writer.Flush()
		inst.writer = nil
	}
}

func cloneMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func truncatePrompt(prompt string, limit int) string {
	trimmed := strings.TrimSpace(prompt)
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "…"
}
