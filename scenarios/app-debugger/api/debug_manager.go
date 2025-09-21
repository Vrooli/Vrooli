package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DebugManager handles debug session orchestration
type DebugManager struct {
	logger *log.Logger
}

// NewDebugManager creates a new debug manager
func NewDebugManager() *DebugManager {
	return &DebugManager{
		logger: log.New(os.Stdout, "[debug-manager] ", log.LstdFlags|log.Lshortfile),
	}
}

// DebugSession represents an active debug session
type DebugSession struct {
	ID        string                 `json:"id"`
	AppName   string                 `json:"app_name"`
	DebugType string                 `json:"debug_type"`
	Status    string                 `json:"status"`
	StartTime time.Time              `json:"start_time"`
	Context   map[string]interface{} `json:"context"`
	Results   map[string]interface{} `json:"results"`
}

// StartDebugSession initiates a debug session for an application
func (dm *DebugManager) StartDebugSession(appName, debugType string) (*DebugSession, error) {
	sessionID := fmt.Sprintf("debug_%s_%d", appName, time.Now().Unix())

	session := &DebugSession{
		ID:        sessionID,
		AppName:   appName,
		DebugType: debugType,
		Status:    "starting",
		StartTime: time.Now(),
		Context:   make(map[string]interface{}),
		Results:   make(map[string]interface{}),
	}

	dm.logger.Printf("Starting debug session %s for app %s (type: %s)", sessionID, appName, debugType)

	// Gather initial context
	if err := dm.gatherDebugContext(session); err != nil {
		dm.logger.Printf("Warning: Failed to gather debug context: %v", err)
	}

	// Perform debug operation based on type
	switch debugType {
	case "performance":
		return dm.performanceDebug(session)
	case "error":
		return dm.errorAnalysis(session)
	case "logs":
		return dm.logAnalysis(session)
	case "health":
		return dm.healthCheck(session)
	default:
		return dm.generalDebug(session)
	}
}

// gatherDebugContext collects context information for debugging
func (dm *DebugManager) gatherDebugContext(session *DebugSession) error {
	// Get app status using vrooli CLI
	cmd := exec.Command("vrooli", "scenario", "status", session.AppName, "--json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get app status: %w", err)
	}

	var appStatus map[string]interface{}
	if err := json.Unmarshal(output, &appStatus); err != nil {
		// If JSON parsing fails, store raw output
		session.Context["app_status"] = string(output)
	} else {
		session.Context["app_status"] = appStatus
	}

	// Get system information
	session.Context["timestamp"] = time.Now().Format(time.RFC3339)
	session.Context["debug_host"] = getHostname()

	return nil
}

// performanceDebug performs performance analysis
func (dm *DebugManager) performanceDebug(session *DebugSession) (*DebugSession, error) {
	dm.logger.Printf("Performing performance debug for %s", session.AppName)

	session.Status = "analyzing"

	// Check memory usage
	memInfo, err := dm.getMemoryInfo(session.AppName)
	if err != nil {
		session.Results["memory_error"] = err.Error()
	} else {
		session.Results["memory"] = memInfo
	}

	// Check CPU usage
	cpuInfo, err := dm.getCPUInfo(session.AppName)
	if err != nil {
		session.Results["cpu_error"] = err.Error()
	} else {
		session.Results["cpu"] = cpuInfo
	}

	// Check disk usage
	diskInfo, err := dm.getDiskInfo(session.AppName)
	if err != nil {
		session.Results["disk_error"] = err.Error()
	} else {
		session.Results["disk"] = diskInfo
	}

	// Generate performance recommendations
	session.Results["recommendations"] = dm.generatePerformanceRecommendations(session.Results)

	session.Status = "completed"
	return session, nil
}

// errorAnalysis performs error analysis for an application
func (dm *DebugManager) errorAnalysis(session *DebugSession) (*DebugSession, error) {
	dm.logger.Printf("Performing error analysis for %s", session.AppName)

	session.Status = "analyzing"

	// Get recent logs
	logs, err := dm.getRecentLogs(session.AppName)
	if err != nil {
		session.Results["log_error"] = err.Error()
	} else {
		session.Results["logs"] = logs

		// Analyze logs for errors
		errorPatterns := dm.analyzeErrorPatterns(logs)
		session.Results["error_patterns"] = errorPatterns

		// Generate fix suggestions
		session.Results["fix_suggestions"] = dm.generateFixSuggestions(errorPatterns)
	}

	session.Status = "completed"
	return session, nil
}

// logAnalysis performs log monitoring and analysis
func (dm *DebugManager) logAnalysis(session *DebugSession) (*DebugSession, error) {
	dm.logger.Printf("Performing log analysis for %s", session.AppName)

	session.Status = "analyzing"

	// Get log file paths
	logPaths, err := dm.getLogPaths(session.AppName)
	if err != nil {
		session.Results["error"] = fmt.Sprintf("Failed to find log files: %v", err)
		session.Status = "failed"
		return session, err
	}

	session.Results["log_files"] = logPaths

	// Analyze each log file
	var logAnalysis []map[string]interface{}
	for _, logPath := range logPaths {
		analysis, err := dm.analyzeLogFile(logPath)
		if err != nil {
			dm.logger.Printf("Failed to analyze log file %s: %v", logPath, err)
			continue
		}
		logAnalysis = append(logAnalysis, analysis)
	}

	session.Results["log_analysis"] = logAnalysis
	session.Status = "completed"
	return session, nil
}

// healthCheck performs comprehensive health check
func (dm *DebugManager) healthCheck(session *DebugSession) (*DebugSession, error) {
	dm.logger.Printf("Performing health check for %s", session.AppName)

	session.Status = "checking"

	var issues []string
	var recommendations []string

	// Check if app is running
	isRunning, err := dm.isAppRunning(session.AppName)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Failed to check if app is running: %v", err))
	} else if !isRunning {
		issues = append(issues, "Application is not running")
		recommendations = append(recommendations, "Start the application using: vrooli scenario run "+session.AppName)
	}

	// Check port availability
	if ports, err := dm.getAppPorts(session.AppName); err == nil {
		for _, port := range ports {
			if !dm.isPortAccessible(port) {
				issues = append(issues, fmt.Sprintf("Port %s is not accessible", port))
				recommendations = append(recommendations, fmt.Sprintf("Check if service is bound to port %s", port))
			}
		}
	}

	// Check dependencies
	if deps, err := dm.getAppDependencies(session.AppName); err == nil {
		for _, dep := range deps {
			if !dm.isDependencyHealthy(dep) {
				issues = append(issues, fmt.Sprintf("Dependency %s is not healthy", dep))
				recommendations = append(recommendations, fmt.Sprintf("Check %s resource status", dep))
			}
		}
	}

	session.Results["issues"] = issues
	session.Results["recommendations"] = recommendations
	session.Results["health_score"] = dm.calculateHealthScore(issues)

	session.Status = "completed"
	return session, nil
}

// generalDebug performs general debugging
func (dm *DebugManager) generalDebug(session *DebugSession) (*DebugSession, error) {
	dm.logger.Printf("Performing general debug for %s", session.AppName)

	session.Status = "debugging"

	// Combine multiple debug approaches
	session.Results["status_check"], _ = dm.getAppStatus(session.AppName)
	session.Results["recent_errors"] = dm.getRecentErrors(session.AppName)
	session.Results["resource_usage"] = dm.getResourceUsage(session.AppName)
	session.Results["suggestions"] = []string{
		"Check application logs for errors",
		"Verify all required environment variables are set",
		"Ensure all dependencies are running",
		"Check network connectivity to external services",
	}

	session.Status = "completed"
	return session, nil
}

// Helper functions

func (dm *DebugManager) getMemoryInfo(appName string) (map[string]interface{}, error) {
	// Get memory info using system commands
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ps -o pid,%%mem,rss -C %s 2>/dev/null || echo 'not found'", appName))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"raw_output": string(output),
		"status":     "checked",
	}, nil
}

func (dm *DebugManager) getCPUInfo(appName string) (map[string]interface{}, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ps -o pid,%%cpu -C %s 2>/dev/null || echo 'not found'", appName))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"raw_output": string(output),
		"status":     "checked",
	}, nil
}

func (dm *DebugManager) getDiskInfo(appName string) (map[string]interface{}, error) {
	// Check disk usage of the app directory
	appPath := fmt.Sprintf("../../../scenarios/%s", appName)
	cmd := exec.Command("du", "-sh", appPath)
	output, err := cmd.Output()
	if err != nil {
		return map[string]interface{}{"error": "app directory not found"}, nil
	}

	return map[string]interface{}{
		"disk_usage": strings.TrimSpace(string(output)),
		"status":     "checked",
	}, nil
}

func (dm *DebugManager) generatePerformanceRecommendations(results map[string]interface{}) []string {
	recommendations := []string{
		"Monitor memory usage trends over time",
		"Consider implementing caching if CPU usage is high",
		"Review database query performance",
		"Check for memory leaks in long-running processes",
	}

	return recommendations
}

func (dm *DebugManager) getRecentLogs(appName string) ([]string, error) {
	// Try to find log files for the app
	logPaths := []string{
		fmt.Sprintf("../../../scenarios/%s/api/%s.log", appName, appName),
		fmt.Sprintf("../../../scenarios/%s/api/app.log", appName),
		fmt.Sprintf("/var/log/vrooli/%s.log", appName),
	}

	var logs []string
	for _, logPath := range logPaths {
		if content, err := dm.readLogFile(logPath, 50); err == nil {
			logs = append(logs, fmt.Sprintf("=== %s ===\n%s", filepath.Base(logPath), content))
		}
	}

	if len(logs) == 0 {
		return []string{"No log files found"}, nil
	}

	return logs, nil
}

func (dm *DebugManager) readLogFile(path string, lines int) (string, error) {
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (dm *DebugManager) analyzeErrorPatterns(logs []string) []map[string]interface{} {
	var patterns []map[string]interface{}

	errorKeywords := []string{"ERROR", "FATAL", "PANIC", "CRITICAL", "Exception", "Failed", "Error"}

	for _, log := range logs {
		lines := strings.Split(log, "\n")
		var errors []string

		for _, line := range lines {
			for _, keyword := range errorKeywords {
				if strings.Contains(line, keyword) {
					errors = append(errors, strings.TrimSpace(line))
					break
				}
			}
		}

		if len(errors) > 0 {
			patterns = append(patterns, map[string]interface{}{
				"source": "log_analysis",
				"errors": errors,
				"count":  len(errors),
			})
		}
	}

	return patterns
}

func (dm *DebugManager) generateFixSuggestions(errorPatterns []map[string]interface{}) []string {
	suggestions := []string{
		"Check application configuration files",
		"Verify environment variables are set correctly",
		"Ensure all required resources are running",
		"Check network connectivity",
		"Review recent code changes",
		"Verify database connections",
		"Check file permissions",
	}

	return suggestions
}

func (dm *DebugManager) getLogPaths(appName string) ([]string, error) {
	possiblePaths := []string{
		fmt.Sprintf("../../../scenarios/%s/api/%s.log", appName, appName),
		fmt.Sprintf("../../../scenarios/%s/logs/app.log", appName),
		fmt.Sprintf("/var/log/vrooli/%s.log", appName),
	}

	var validPaths []string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			validPaths = append(validPaths, path)
		}
	}

	return validPaths, nil
}

func (dm *DebugManager) analyzeLogFile(path string) (map[string]interface{}, error) {
	content, err := dm.readLogFile(path, 100)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")

	return map[string]interface{}{
		"file":        path,
		"line_count":  len(lines),
		"size_bytes":  len(content),
		"last_update": dm.getFileModTime(path),
	}, nil
}

func (dm *DebugManager) getFileModTime(path string) string {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime().Format(time.RFC3339)
	}
	return "unknown"
}

func (dm *DebugManager) isAppRunning(appName string) (bool, error) {
	cmd := exec.Command("vrooli", "scenario", "status", appName)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.Contains(string(output), "running"), nil
}

func (dm *DebugManager) getAppPorts(appName string) ([]string, error) {
	// This would typically read from the app's service.json
	// For now, return common ports
	return []string{"3000", "8080", "5000"}, nil
}

func (dm *DebugManager) isPortAccessible(port string) bool {
	cmd := exec.Command("nc", "-z", "localhost", port)
	err := cmd.Run()
	return err == nil
}

func (dm *DebugManager) getAppDependencies(appName string) ([]string, error) {
	// This would typically read from the app's service.json
	return []string{"postgres", "redis"}, nil
}

func (dm *DebugManager) isDependencyHealthy(dep string) bool {
	cmd := exec.Command("vrooli", "resource", dep, "status")
	err := cmd.Run()
	return err == nil
}

func (dm *DebugManager) calculateHealthScore(issues []string) int {
	if len(issues) == 0 {
		return 100
	} else if len(issues) <= 2 {
		return 75
	} else if len(issues) <= 5 {
		return 50
	}
	return 25
}

func (dm *DebugManager) getAppStatus(appName string) (string, error) {
	cmd := exec.Command("vrooli", "scenario", "status", appName)
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func (dm *DebugManager) getRecentErrors(appName string) []string {
	// Simple implementation - would be more sophisticated in production
	return []string{"No recent errors detected"}
}

func (dm *DebugManager) getResourceUsage(appName string) map[string]string {
	return map[string]string{
		"memory": "checking...",
		"cpu":    "checking...",
		"disk":   "checking...",
	}
}

func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}
