package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/system-monitor/internal/config"
	"github.com/vrooli/system-monitor/internal/models"
	"github.com/vrooli/system-monitor/internal/repository"
)

// InvestigationService handles anomaly investigations
type InvestigationService struct {
	config   *config.Config
	repo     repository.InvestigationRepository
	alertSvc *AlertService
	mu       sync.RWMutex
}

// NewInvestigationService creates a new investigation service
func NewInvestigationService(cfg *config.Config, repo repository.InvestigationRepository, alertSvc *AlertService) *InvestigationService {
	return &InvestigationService{
		config:   cfg,
		repo:     repo,
		alertSvc: alertSvc,
	}
}

// TriggerInvestigation starts a new investigation
func (s *InvestigationService) TriggerInvestigation(ctx context.Context) (*models.Investigation, error) {
	// Generate investigation ID
	investigationID := fmt.Sprintf("inv_%d", time.Now().Unix())
	
	// Create investigation
	investigation := &models.Investigation{
		ID:        investigationID,
		Status:    "queued",
		AnomalyID: fmt.Sprintf("ai_investigation_%d", time.Now().Unix()),
		StartTime: time.Now(),
		Findings:  "Investigation queued for processing...",
		Progress:  0,
		Details:   make(map[string]interface{}),
		Steps:     []models.InvestigationStep{},
	}
	
	// Save to repository
	if err := s.repo.CreateInvestigation(ctx, investigation); err != nil {
		return nil, fmt.Errorf("failed to create investigation: %w", err)
	}
	
	// Start investigation in background
	go s.runInvestigation(investigationID)
	
	return investigation, nil
}

// GetInvestigation retrieves an investigation by ID
func (s *InvestigationService) GetInvestigation(ctx context.Context, id string) (*models.Investigation, error) {
	return s.repo.GetInvestigation(ctx, id)
}

// GetLatestInvestigation retrieves the most recent investigation
func (s *InvestigationService) GetLatestInvestigation(ctx context.Context) (*models.Investigation, error) {
	investigation, err := s.repo.GetLatestInvestigation(ctx)
	if err != nil {
		// Return default if none exists
		return &models.Investigation{
			ID:        "inv_default",
			Status:    "pending",
			AnomalyID: "none",
			StartTime: time.Now(),
			Findings:  "No investigations have been triggered yet. Click 'RUN ANOMALY CHECK' to start an investigation.",
		}, nil
	}
	return investigation, nil
}

// UpdateInvestigationStatus updates the status of an investigation
func (s *InvestigationService) UpdateInvestigationStatus(ctx context.Context, id string, status string) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		return err
	}
	
	investigation.Status = status
	if status == "completed" || status == "failed" {
		now := time.Now()
		investigation.EndTime = &now
	}
	
	return s.repo.UpdateInvestigation(ctx, investigation)
}

// UpdateInvestigationFindings updates the findings of an investigation
func (s *InvestigationService) UpdateInvestigationFindings(ctx context.Context, id string, findings string, details map[string]interface{}) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		return err
	}
	
	investigation.Findings = findings
	if details != nil {
		for k, v := range details {
			investigation.Details[k] = v
		}
	}
	
	return s.repo.UpdateInvestigation(ctx, investigation)
}

// UpdateInvestigationProgress updates the progress of an investigation
func (s *InvestigationService) UpdateInvestigationProgress(ctx context.Context, id string, progress int) error {
	investigation, err := s.repo.GetInvestigation(ctx, id)
	if err != nil {
		return err
	}
	
	investigation.Progress = progress
	return s.repo.UpdateInvestigation(ctx, investigation)
}

// AddInvestigationStep adds a step to an investigation
func (s *InvestigationService) AddInvestigationStep(ctx context.Context, id string, step models.InvestigationStep) error {
	step.StartTime = time.Now()
	return s.repo.SaveInvestigationStep(ctx, id, &step)
}

// runInvestigation performs the actual investigation
func (s *InvestigationService) runInvestigation(investigationID string) {
	ctx := context.Background()
	
	// Update status to in_progress
	s.UpdateInvestigationStatus(ctx, investigationID, "in_progress")
	s.UpdateInvestigationProgress(ctx, investigationID, 10)
	
	// Collect current metrics for context
	cpuUsage := s.getCPUUsage()
	memoryUsage := s.getMemoryUsage()
	tcpConnections := s.getTCPConnections()
	timestamp := time.Now().Format(time.RFC3339)
	
	// Try Claude Code first, then fallback to basic analysis
	findings, details := s.performInvestigation(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp)
	
	// Update investigation with findings
	s.UpdateInvestigationFindings(ctx, investigationID, findings, details)
	s.UpdateInvestigationProgress(ctx, investigationID, 100)
	s.UpdateInvestigationStatus(ctx, investigationID, "completed")
}

// performInvestigation executes the investigation logic
func (s *InvestigationService) performInvestigation(investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp string) (string, map[string]interface{}) {
	// Try Claude Code integration
	prompt := s.buildInvestigationPrompt(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp)
	
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`cd ${VROOLI_ROOT:-${HOME}/Vrooli} && echo %q | timeout 10 vrooli resource claude-code run 2>&1 || true`, prompt))
	output, err := cmd.Output()
	
	// Check if Claude Code worked
	if err == nil && !strings.Contains(string(output), "USAGE:") && !strings.Contains(string(output), "Failed to load library") {
		return string(output), map[string]interface{}{
			"source":     "claude_code",
			"risk_level": "low",
		}
	}
	
	// Fallback to basic analysis
	return s.performBasicAnalysis(cpuUsage, memoryUsage, tcpConnections, timestamp, investigationID)
}

// performBasicAnalysis performs a basic system analysis
func (s *InvestigationService) performBasicAnalysis(cpuUsage, memoryUsage float64, tcpConnections int, timestamp, investigationID string) (string, map[string]interface{}) {
	riskLevel := "low"
	anomalies := []string{}
	recommendations := []string{}
	
	// Analyze CPU usage
	if cpuUsage > 80 {
		riskLevel = "high"
		anomalies = append(anomalies, fmt.Sprintf("High CPU usage: %.2f%%", cpuUsage))
		recommendations = append(recommendations, "Investigate top CPU-consuming processes")
	} else if cpuUsage > 60 {
		riskLevel = "medium"
		anomalies = append(anomalies, fmt.Sprintf("Elevated CPU usage: %.2f%%", cpuUsage))
	}
	
	// Analyze memory usage
	if memoryUsage > 90 {
		if riskLevel == "low" {
			riskLevel = "high"
		}
		anomalies = append(anomalies, fmt.Sprintf("Critical memory usage: %.2f%%", memoryUsage))
		recommendations = append(recommendations, "Check for memory leaks")
	} else if memoryUsage > 75 {
		if riskLevel == "low" {
			riskLevel = "medium"
		}
		anomalies = append(anomalies, fmt.Sprintf("High memory usage: %.2f%%", memoryUsage))
	}
	
	// Analyze network connections
	if tcpConnections > 500 {
		if riskLevel == "low" {
			riskLevel = "medium"
		}
		anomalies = append(anomalies, fmt.Sprintf("High number of TCP connections: %d", tcpConnections))
		recommendations = append(recommendations, "Review network connections")
	}
	
	// Build findings report
	findings := fmt.Sprintf(`### Investigation Summary

**Status**: %s
**Investigation ID**: %s
**Timestamp**: %s

**Key Findings**:
- CPU Usage: %.2f%%
- Memory Usage: %.2f%%
- TCP Connections: %d

**Anomalies Detected**: %d
%s

**Risk Level**: %s

**Recommendations**:
%s`,
		func() string {
			if len(anomalies) > 0 {
				return "Warning"
			}
			return "Normal"
		}(),
		investigationID,
		timestamp,
		cpuUsage,
		memoryUsage,
		tcpConnections,
		len(anomalies),
		func() string {
			if len(anomalies) == 0 {
				return "- No anomalies detected"
			}
			result := ""
			for _, a := range anomalies {
				result += fmt.Sprintf("- %s\n", a)
			}
			return result
		}(),
		riskLevel,
		func() string {
			if len(recommendations) == 0 {
				return "- Continue normal monitoring"
			}
			result := ""
			for _, r := range recommendations {
				result += fmt.Sprintf("- %s\n", r)
			}
			return result
		}(),
	)
	
	details := map[string]interface{}{
		"source":            "fallback_analysis",
		"risk_level":        riskLevel,
		"anomalies_found":   len(anomalies),
		"recommendations_count": len(recommendations),
		"critical_issues":   riskLevel == "high",
	}
	
	return findings, details
}

// buildInvestigationPrompt builds the prompt for Claude Code
func (s *InvestigationService) buildInvestigationPrompt(investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp string) string {
	// Try to load prompt template
	prompt, err := s.loadPromptTemplate(investigationID, cpuUsage, memoryUsage, tcpConnections, timestamp)
	if err != nil {
		// Use default prompt
		return fmt.Sprintf(`System Anomaly Investigation
Investigation ID: %s
API Base URL: http://localhost:8080
CPU: %.2f%%, Memory: %.2f%%, TCP Connections: %d
Analyze system for anomalies and provide findings.`,
			investigationID, cpuUsage, memoryUsage, tcpConnections)
	}
	return prompt
}

// loadPromptTemplate loads and processes the prompt template
func (s *InvestigationService) loadPromptTemplate(investigationID string, cpuUsage, memoryUsage float64, tcpConnections int, timestamp string) (string, error) {
	// Get VROOLI_ROOT
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/root"
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}
	
	// Try to load prompt file
	promptPaths := []string{
		filepath.Join(vrooliRoot, "scenarios", "system-monitor", "initialization", "claude-code", "anomaly-check.md"),
		filepath.Join("initialization", "claude-code", "anomaly-check.md"),
	}
	
	var promptContent []byte
	var err error
	
	for _, path := range promptPaths {
		promptContent, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	
	if err != nil {
		return "", fmt.Errorf("could not read prompt file: %v", err)
	}
	
	// Replace placeholders
	prompt := string(promptContent)
	prompt = strings.ReplaceAll(prompt, "{{CPU_USAGE}}", fmt.Sprintf("%.2f", cpuUsage))
	prompt = strings.ReplaceAll(prompt, "{{MEMORY_USAGE}}", fmt.Sprintf("%.2f", memoryUsage))
	prompt = strings.ReplaceAll(prompt, "{{TCP_CONNECTIONS}}", strconv.Itoa(tcpConnections))
	prompt = strings.ReplaceAll(prompt, "{{TIMESTAMP}}", timestamp)
	prompt = strings.ReplaceAll(prompt, "{{INVESTIGATION_ID}}", investigationID)
	prompt = strings.ReplaceAll(prompt, "{{API_BASE_URL}}", "http://localhost:8080")
	
	return prompt, nil
}

// Helper methods to get system metrics
func (s *InvestigationService) getCPUUsage() float64 {
	cmd := exec.Command("bash", "-c", "grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$3+$4+$5)} END {print usage}'")
	output, err := cmd.Output()
	if err != nil {
		return 15.0 // Default value
	}
	
	usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 15.0
	}
	return usage
}

func (s *InvestigationService) getMemoryUsage() float64 {
	cmd := exec.Command("bash", "-c", "free | grep Mem | awk '{print ($3/$2) * 100.0}'")
	output, err := cmd.Output()
	if err != nil {
		return 45.0 // Default value
	}
	
	usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 45.0
	}
	return usage
}

func (s *InvestigationService) getTCPConnections() int {
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep ESTABLISHED | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 50 // Default value
	}
	
	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 50
	}
	return count
}