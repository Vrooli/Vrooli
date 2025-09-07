package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Resource-specific helper functions for v2.0 compliance and health checking

// Check v2.0 contract compliance for a resource
func checkV2Compliance(resourceName string) float64 {
	resourcesDir := getEnv("RESOURCES_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/resources"))
	resourcePath := filepath.Join(resourcesDir, resourceName)
	
	score := 0.0
	maxScore := 15.0 // Increased for more detailed scoring
	
	// Check if resource directory exists
	if _, err := os.Stat(resourcePath); err != nil {
		log.Printf("Resource directory not found: %s (path: %s)", resourceName, resourcePath)
		return 0.0 // Resource doesn't exist
	}
	
	// Check for lib/ directory (required for v2.0)
	libPath := filepath.Join(resourcePath, "lib")
	if _, err := os.Stat(libPath); err == nil {
		score += 1.0
		
		// Enhanced lib file validation with content parsing
		score += checkLibFileImplementations(libPath)
	}
	
	// Enhanced service.json validation
	serviceJsonScore := validateServiceJson(filepath.Join(resourcePath, "service.json"))
	score += serviceJsonScore
	
	// Check for README.md with content validation
	readmeScore := validateReadmeContent(filepath.Join(resourcePath, "README.md"))
	score += readmeScore
	
	// Enhanced health check validation
	healthScore := validateHealthImplementation(resourcePath, libPath)
	score += healthScore
	
	// Check for required lifecycle hooks
	lifecycleScore := validateLifecycleHooks(libPath)
	score += lifecycleScore
	
	// Convert to percentage
	percentage := (score / maxScore) * 100.0
	if percentage > 100.0 {
		percentage = 100.0
	}
	
	log.Printf("v2.0 compliance for %s: %.1f%% (%.1f/%d points)", resourceName, percentage, score, int(maxScore))
	return percentage
}

// Enhanced health check reliability for a resource with comprehensive validation
func checkHealthReliability(resourceName string) float64 {
	resourcesDir := getEnv("RESOURCES_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/resources"))
	resourcePath := filepath.Join(resourcesDir, resourceName)
	
	totalScore := 0.0
	maxScore := 10.0
	
	// 1. Basic health script execution (30% of score)
	basicHealthScore := checkBasicHealthExecution(resourcePath, resourceName)
	totalScore += basicHealthScore * 3.0
	
	// 2. Network connectivity tests (25% of score) 
	networkScore := checkNetworkConnectivity(resourcePath, resourceName)
	totalScore += networkScore * 2.5
	
	// 3. Service response validation (25% of score)
	responseScore := validateServiceResponses(resourcePath, resourceName)
	totalScore += responseScore * 2.5
	
	// 4. Dependency checking (20% of score)
	dependencyScore := checkServiceDependencies(resourcePath, resourceName)
	totalScore += dependencyScore * 2.0
	
	reliability := (totalScore / maxScore) * 100.0
	if reliability > 100.0 {
		reliability = 100.0
	}
	
	log.Printf("Health reliability for %s: %.1f%% (%.1f/%d points)", resourceName, reliability, totalScore, int(maxScore))
	return reliability
}

// Basic health script execution testing
func checkBasicHealthExecution(resourcePath, resourceName string) float64 {
	healthScripts := []string{
		filepath.Join(resourcePath, "lib", "health.sh"),
		filepath.Join(resourcePath, "health.sh"),
		fmt.Sprintf("resource-%s", resourceName), // Try CLI command
	}
	
	successCount := 0
	totalAttempts := 3
	
	for attempt := 0; attempt < totalAttempts; attempt++ {
		for _, healthScript := range healthScripts {
			if strings.HasPrefix(healthScript, "resource-") {
				// Try CLI health command with timeout
				cmd := exec.Command(healthScript, "health")
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				
				done := make(chan error, 1)
				go func() { done <- cmd.Run() }()
				
				select {
				case err := <-done:
					if err == nil {
						successCount++
						goto nextAttempt
					}
				case <-time.After(10 * time.Second): // 10 second timeout
					syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
					log.Printf("Health check timeout for %s", healthScript)
				}
			} else {
				// Try script file with timeout
				if _, err := os.Stat(healthScript); err == nil {
					cmd := exec.Command("bash", healthScript)
					cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
					
					done := make(chan error, 1)
					go func() { done <- cmd.Run() }()
					
					select {
					case err := <-done:
						if err == nil {
							successCount++
							goto nextAttempt
						}
					case <-time.After(10 * time.Second): // 10 second timeout
						syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
						log.Printf("Health script timeout for %s", healthScript)
					}
				}
			}
		}
		nextAttempt:
		time.Sleep(200 * time.Millisecond) // Brief delay between attempts
	}
	
	score := float64(successCount) / float64(totalAttempts)
	log.Printf("  Basic health execution: %d/%d successful", successCount, totalAttempts)
	return score
}

// Network connectivity testing for resource
func checkNetworkConnectivity(resourcePath, resourceName string) float64 {
	score := 0.0
	maxTests := 4.0
	
	// Load service configuration to get ports
	serviceConfig := loadServiceConfig(filepath.Join(resourcePath, "service.json"))
	
	// Test 1: Port availability check
	if ports := getResourcePorts(serviceConfig); len(ports) > 0 {
		for _, port := range ports {
			if isPortReachable("localhost", port, 5*time.Second) {
				score += 0.5
				log.Printf("  Port %d: reachable", port)
			} else {
				log.Printf("  Port %d: not reachable", port)
			}
		}
		if score > 1.0 { score = 1.0 } // Cap port checks at 1.0 point
	}
	
	// Test 2: DNS resolution (if applicable)
	if hostname := getResourceHostname(serviceConfig); hostname != "" && hostname != "localhost" {
		if isDNSResolvable(hostname) {
			score += 0.25
			log.Printf("  DNS resolution: %s resolved", hostname)
		} else {
			log.Printf("  DNS resolution: %s failed", hostname)
		}
	} else {
		score += 0.25 // Skip test for localhost resources
	}
	
	// Test 3: Network interface check
	if hasValidNetworkInterface() {
		score += 0.25
		log.Printf("  Network interface: available")
	} else {
		log.Printf("  Network interface: issues detected")
	}
	
	// Test 4: Firewall/routing check
	if checkBasicRouting() {
		score += 0.25
		log.Printf("  Basic routing: functional")
	} else {
		log.Printf("  Basic routing: issues detected")
	}
	
	return score / maxTests
}

// Validate service responses beyond just exit codes
func validateServiceResponses(resourcePath, resourceName string) float64 {
	score := 0.0
	maxTests := 4.0
	
	serviceConfig := loadServiceConfig(filepath.Join(resourcePath, "service.json"))
	ports := getResourcePorts(serviceConfig)
	
	if len(ports) == 0 {
		// For services without HTTP endpoints, validate CLI responses
		return validateCLIResponses(resourceName) / maxTests
	}
	
	// Test HTTP endpoints if available
	for _, port := range ports {
		baseURL := fmt.Sprintf("http://localhost:%d", port)
		
		// Test 1: Health endpoint response
		if validateHTTPHealthEndpoint(baseURL) {
			score += 1.0
			log.Printf("  HTTP health endpoint: valid response")
		} else {
			log.Printf("  HTTP health endpoint: invalid or missing")
		}
		
		// Test 2: Status endpoint response  
		if validateHTTPStatusEndpoint(baseURL) {
			score += 1.0
			log.Printf("  HTTP status endpoint: valid response")
		} else {
			log.Printf("  HTTP status endpoint: invalid or missing")
		}
		
		// Test 3: Response format validation
		if validateResponseFormat(baseURL) {
			score += 1.0 
			log.Printf("  Response format: valid JSON/expected format")
		} else {
			log.Printf("  Response format: invalid or unexpected")
		}
		
		// Test 4: Response time validation
		if validateResponseTime(baseURL) {
			score += 1.0
			log.Printf("  Response time: acceptable (<5s)")
		} else {
			log.Printf("  Response time: too slow (>5s)")
		}
		
		break // Only test the first port for now
	}
	
	return score / maxTests
}

// Check service dependencies
func checkServiceDependencies(resourcePath, resourceName string) float64 {
	score := 0.0
	maxTests := 3.0
	
	serviceConfig := loadServiceConfig(filepath.Join(resourcePath, "service.json"))
	
	// Test 1: Check declared dependencies
	if deps := getDeclaredDependencies(serviceConfig); len(deps) > 0 {
		availableDeps := 0
		for _, dep := range deps {
			if isDependencyAvailable(dep) {
				availableDeps++
			}
		}
		score += float64(availableDeps) / float64(len(deps))
		log.Printf("  Dependencies: %d/%d available", availableDeps, len(deps))
	} else {
		score += 1.0 // No dependencies means this test passes
		log.Printf("  Dependencies: none declared")
	}
	
	// Test 2: Check common system dependencies
	systemDeps := []string{"curl", "bash", "ps", "grep"}
	availableSystemDeps := 0
	for _, dep := range systemDeps {
		if _, err := exec.LookPath(dep); err == nil {
			availableSystemDeps++
		}
	}
	score += float64(availableSystemDeps) / float64(len(systemDeps))
	log.Printf("  System dependencies: %d/%d available", availableSystemDeps, len(systemDeps))
	
	// Test 3: Check resource-specific requirements
	if checkResourceSpecificDependencies(resourceName) {
		score += 1.0
		log.Printf("  Resource-specific deps: all satisfied")
	} else {
		log.Printf("  Resource-specific deps: some missing")
	}
	
	return score / maxTests
}

// Check CLI coverage for a resource
func checkCLICoverage(resourceName string) float64 {
	cliName := fmt.Sprintf("resource-%s", resourceName)
	
	// Check if CLI exists
	if _, err := exec.LookPath(cliName); err != nil {
		return 0.0
	}
	
	score := 0.0
	maxScore := 5.0
	
	// Check for standard commands
	standardCommands := []string{
		"status",
		"health", 
		"help",
		"logs",
		"content",
	}
	
	for _, command := range standardCommands {
		cmd := exec.Command(cliName, command, "--help")
		if err := cmd.Run(); err == nil {
			score += 1.0
		}
	}
	
	percentage := (score / maxScore) * 100.0
	return percentage
}

// Check documentation completeness for a resource
func checkDocumentationCompleteness(resourceName string) float64 {
	resourcesDir := getEnv("RESOURCES_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/resources"))
	resourcePath := filepath.Join(resourcesDir, resourceName)
	
	score := 0.0
	maxScore := 5.0
	
	// Check for README.md
	readmePath := filepath.Join(resourcePath, "README.md")
	if content, err := os.ReadFile(readmePath); err == nil {
		readmeContent := strings.ToLower(string(content))
		
		// Check for required sections
		requiredSections := []string{
			"usage",
			"configuration",
			"environment",
			"troubleshooting",
			"example",
		}
		
		for _, section := range requiredSections {
			if strings.Contains(readmeContent, section) {
				score += 1.0
			}
		}
	}
	
	percentage := (score / maxScore) * 100.0
	return percentage
}

// Get list of available resources for improvement
func getAvailableResources() []ResourceHealth {
	resourcesDir := getEnv("RESOURCES_DIR", filepath.Join(os.Getenv("HOME"), "Vrooli/resources"))
	
	var resources []ResourceHealth
	
	// List directories in resources folder
	if dirs, err := os.ReadDir(resourcesDir); err == nil {
		for _, dir := range dirs {
			if dir.IsDir() {
				resourceName := dir.Name()
				
				// Skip hidden directories
				if strings.HasPrefix(resourceName, ".") {
					continue
				}
				
				// Get compliance scores
				v2Score := checkV2Compliance(resourceName)
				healthScore := checkHealthReliability(resourceName)
				
				// Determine status
				status := "healthy"
				health := "healthy"
				if v2Score < 80.0 {
					status = "needs-improvement"
					health = "warning"
				}
				if v2Score < 50.0 || healthScore < 70.0 {
					status = "unhealthy"
					health = "unhealthy"
				}
				
				resource := ResourceHealth{
					Name:              resourceName,
					Status:            status,
					Health:            health,
					V2ComplianceScore: v2Score,
					HealthReliability: healthScore,
					IssueCount:        calculateIssueCount(v2Score, healthScore),
				}
				
				resources = append(resources, resource)
			}
		}
	}
	
	return resources
}

// Calculate issue count based on scores
func calculateIssueCount(v2Score, healthScore float64) int {
	issueCount := 0
	
	if v2Score < 90.0 {
		issueCount++
	}
	if v2Score < 70.0 {
		issueCount++
	}
	if healthScore < 95.0 {
		issueCount++
	}
	if healthScore < 80.0 {
		issueCount++
	}
	
	return issueCount
}

// Convert resource reports to improvement queue items
func resourceReportToImprovements(reports []ResourceReport) []QueueItem {
	var improvements []QueueItem
	
	for _, report := range reports {
		improvement := QueueItem{
			ID:          fmt.Sprintf("resource-%s-%d", report.ResourceName, time.Now().UnixNano()),
			Title:       fmt.Sprintf("Fix %s issue in %s", report.IssueType, report.ResourceName),
			Description: fmt.Sprintf("Issue: %s\nDetails: %s", report.IssueType, report.Description),
			Type:        "fix",
			Target:      report.ResourceName,
			Priority:    mapSeverityToPriority(report.Severity),
			CreatedBy:   "resource-monitor",
			CreatedAt:   time.Now(),
			Estimates: map[string]interface{}{
				"impact":       7,
				"urgency":      "high",
				"success_prob": 0.8,
				"resource_cost": "moderate",
			},
		}
		improvements = append(improvements, improvement)
	}
	
	return improvements
}

// Convert metrics to improvement queue items
func metricsToImprovements(metrics *ResourceMetrics) []QueueItem {
	var improvements []QueueItem
	
	// Generate improvements based on low scores
	if metrics.V2ComplianceScore < 90.0 {
		improvement := QueueItem{
			ID:          fmt.Sprintf("v2-compliance-%s-%d", metrics.ResourceName, time.Now().UnixNano()),
			Title:       fmt.Sprintf("Improve v2.0 compliance for %s", metrics.ResourceName),
			Description: fmt.Sprintf("Current v2.0 compliance score: %.1f%%. Needs improvement to meet 90%% threshold.", metrics.V2ComplianceScore),
			Type:        "v2-compliance",
			Target:      metrics.ResourceName,
			Priority:    "high",
			CreatedBy:   "metrics-monitor",
			CreatedAt:   time.Now(),
			Estimates: map[string]interface{}{
				"impact":       8,
				"urgency":      "high",
				"success_prob": 0.9,
				"resource_cost": "moderate",
			},
		}
		improvements = append(improvements, improvement)
	}
	
	if metrics.HealthReliability < 95.0 {
		improvement := QueueItem{
			ID:          fmt.Sprintf("health-reliability-%s-%d", metrics.ResourceName, time.Now().UnixNano()),
			Title:       fmt.Sprintf("Improve health check reliability for %s", metrics.ResourceName),
			Description: fmt.Sprintf("Current health reliability: %.1f%%. Needs improvement to meet 95%% threshold.", metrics.HealthReliability),
			Type:        "health-check",
			Target:      metrics.ResourceName,
			Priority:    "high",
			CreatedBy:   "metrics-monitor",
			CreatedAt:   time.Now(),
			Estimates: map[string]interface{}{
				"impact":       7,
				"urgency":      "medium",
				"success_prob": 0.8,
				"resource_cost": "low",
			},
		}
		improvements = append(improvements, improvement)
	}
	
	if metrics.CLICoverage < 80.0 {
		improvement := QueueItem{
			ID:          fmt.Sprintf("cli-coverage-%s-%d", metrics.ResourceName, time.Now().UnixNano()),
			Title:       fmt.Sprintf("Improve CLI coverage for %s", metrics.ResourceName),
			Description: fmt.Sprintf("Current CLI coverage: %.1f%%. Add missing standard commands.", metrics.CLICoverage),
			Type:        "cli-enhancement",
			Target:      metrics.ResourceName,
			Priority:    "medium",
			CreatedBy:   "metrics-monitor",
			CreatedAt:   time.Now(),
			Estimates: map[string]interface{}{
				"impact":       6,
				"urgency":      "medium",
				"success_prob": 0.8,
				"resource_cost": "moderate",
			},
		}
		improvements = append(improvements, improvement)
	}
	
	return improvements
}

// Map severity to priority
func mapSeverityToPriority(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "high":
		return "critical"
	case "medium":
		return "high"
	case "low":
		return "medium"
	default:
		return "medium"
	}
}

// Determine overall resource status based on metrics
func determineResourceStatus(metrics *ResourceMetrics) string {
	if metrics.V2ComplianceScore >= 90.0 && metrics.HealthReliability >= 95.0 {
		return "excellent"
	}
	if metrics.V2ComplianceScore >= 80.0 && metrics.HealthReliability >= 90.0 {
		return "good"
	}
	if metrics.V2ComplianceScore >= 70.0 && metrics.HealthReliability >= 80.0 {
		return "needs-improvement"
	}
	return "poor"
}

// Parse recommendations from AI response
func parseRecommendations(response string) []string {
	// Try to extract recommendations from Claude's response
	lines := strings.Split(response, "\n")
	var recommendations []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for recommendation patterns
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "recommend") ||
		   strings.Contains(lowerLine, "suggest") ||
		   strings.Contains(lowerLine, "improve") ||
		   strings.Contains(lowerLine, "add") ||
		   strings.Contains(lowerLine, "implement") {
			if len(line) > 10 && len(line) < 200 {
				recommendations = append(recommendations, line)
			}
		}
	}
	
	// Try JSON parsing if no recommendations found
	if len(recommendations) == 0 {
		var jsonArray []string
		if err := json.Unmarshal([]byte(response), &jsonArray); err == nil {
			recommendations = jsonArray
		}
	}
	
	// Fallback recommendations
	if len(recommendations) == 0 {
		recommendations = []string{
			"Review v2.0 contract compliance",
			"Implement robust health checking",
			"Standardize CLI commands",
		}
	}
	
	return recommendations
}

// Search Vrooli memory for improvement patterns
func searchVrooliMemory(target, improvementType string) (string, error) {
	// Try to use resource-qdrant CLI if available
	cmd := exec.Command("resource-qdrant", "search", "--collection", "resource-improvements", "--query", fmt.Sprintf("%s %s", target, improvementType), "--limit", "5")
	output, err := cmd.Output()
	if err == nil {
		// Parse the search results and extract relevant context
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var context []string
		for _, line := range lines {
			if strings.Contains(line, "text:") {
				// Extract text content
				if idx := strings.Index(line, "text:"); idx >= 0 {
					text := strings.TrimSpace(line[idx+5:])
					if text != "" {
						context = append(context, text)
					}
				}
			}
		}
		if len(context) > 0 {
			log.Printf("Found %d relevant memory entries for %s/%s", len(context), target, improvementType)
			return strings.Join(context, "\n\n"), nil
		}
	} else {
		log.Printf("Qdrant search failed for %s/%s: %v", target, improvementType, err)
	}
	
	// Fallback: Search local improvement history files
	historyDir := "../queue/completed"
	files, err := filepath.Glob(filepath.Join(historyDir, "*.yaml"))
	if err != nil {
		return "No relevant memory context found", nil
	}
	
	var relevantContext []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		// Check if file content is relevant to target/type
		contentStr := strings.ToLower(string(content))
		if strings.Contains(contentStr, strings.ToLower(target)) || strings.Contains(contentStr, strings.ToLower(improvementType)) {
			// Extract relevant lines
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "description") && strings.Contains(line, ":") {
					if parts := strings.SplitN(line, ":", 2); len(parts) > 1 {
						relevantContext = append(relevantContext, strings.TrimSpace(parts[1]))
					}
				}
			}
			if len(relevantContext) >= 3 {
				break
			}
		}
	}
	
	if len(relevantContext) > 0 {
		return "Previous related improvements:\n" + strings.Join(relevantContext, "\n"), nil
	}
	
	return "No relevant memory context found", nil
}

// Update Vrooli memory with improvement results
func updateVrooliMemory(result *ImprovementResult) {
	// Prepare memory entry for storage
	memoryEntry := map[string]interface{}{
		"queue_item_id": result.QueueItemID,
		"success":       result.Success,
		"changes":       strings.Join(result.Changes, "; "),
		"test_results":  strings.Join(result.TestResults, "; "),
		"completed_at":  result.CompletedAt.Format(time.RFC3339),
	}
	
	// If failure, include failure reason
	if !result.Success {
		memoryEntry["failure_reason"] = result.FailureReason
	}
	
	// Try to store in Qdrant using CLI
	memoryJSON, _ := json.Marshal(memoryEntry)
	cmd := exec.Command("resource-qdrant", "upsert", "--collection", "resource-improvements", "--payload", string(memoryJSON))
	if err := cmd.Run(); err != nil {
		// Fallback: Store locally in a memory file
		memoryFile := "../data/resource_improvement_memory.jsonl"
		file, err := os.OpenFile(memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to store memory: %v", err)
			return
		}
		defer file.Close()
		
		// Add timestamp and write as JSONL
		memoryEntry["stored_at"] = time.Now().Format(time.RFC3339)
		memoryJSON, _ = json.Marshal(memoryEntry)
		file.WriteString(string(memoryJSON) + "\n")
	}
	
	log.Printf("Memory updated for improvement %s (success: %v)", result.QueueItemID, result.Success)
}

// Load queue item by ID (referenced but needs to be defined)
func loadQueueItemByID(id string) (*QueueItem, error) {
	statuses := []string{"pending", "in-progress", "completed", "failed"}
	
	for _, status := range statuses {
		files, err := filepath.Glob(filepath.Join(queueDir, status, fmt.Sprintf("%s*.yaml", id)))
		if err != nil {
			continue
		}
		
		for _, file := range files {
			item, err := loadQueueItem(file)
			if err == nil && item.ID == id {
				return item, nil
			}
		}
	}
	
	return nil, fmt.Errorf("queue item %s not found", id)
}

// Helper function to count passed gates
func countPassedGates(gates []ValidationGate) int {
	count := 0
	for _, gate := range gates {
		if gate.Passed {
			count++
		}
	}
	return count
}

// Enhanced lib file validation with content parsing
func checkLibFileImplementations(libPath string) float64 {
	score := 0.0
	
	// Required lib files with their essential functions
	requiredLibFiles := map[string][]string{
		"common.sh":    {"get_env", "log_info", "log_error", "is_port_available"},
		"health.sh":    {"check_health", "check_dependencies", "health_status"},
		"lifecycle.sh": {"start_service", "stop_service", "restart_service", "service_status"},
		"content.sh":   {"list_content", "add_content", "remove_content", "validate_content"},
	}
	
	for fileName, requiredFunctions := range requiredLibFiles {
		filePath := filepath.Join(libPath, fileName)
		if content, err := os.ReadFile(filePath); err == nil {
			score += 0.5 // File exists
			
			fileContent := string(content)
			functionsFound := 0
			
			for _, requiredFunc := range requiredFunctions {
				// Check for function definition patterns
				patterns := []string{
					fmt.Sprintf(`%s\s*\(\s*\)\s*\{`, requiredFunc),
					fmt.Sprintf(`function\s+%s\s*\(\s*\)`, requiredFunc),
					fmt.Sprintf(`%s\s*=\s*function`, requiredFunc),
				}
				
				found := false
				for _, pattern := range patterns {
					if matched, _ := regexp.MatchString(pattern, fileContent); matched {
						found = true
						break
					}
				}
				
				if found {
					functionsFound++
				}
			}
			
			// Award points based on function completion ratio
			if len(requiredFunctions) > 0 {
				score += (float64(functionsFound) / float64(len(requiredFunctions))) * 1.5
			}
			
			log.Printf("  %s: %d/%d required functions found", fileName, functionsFound, len(requiredFunctions))
		} else {
			log.Printf("  %s: missing", fileName)
		}
	}
	
	return score
}

// Validate service.json schema and content
func validateServiceJson(serviceJsonPath string) float64 {
	score := 0.0
	
	if content, err := os.ReadFile(serviceJsonPath); err == nil {
		score += 0.5 // File exists
		
		var serviceConfig map[string]interface{}
		if err := json.Unmarshal(content, &serviceConfig); err == nil {
			score += 0.5 // Valid JSON
			
			// Check for required v2.0 fields
			requiredFields := []string{"name", "version", "type", "ports", "health", "lifecycle"}
			fieldsFound := 0
			
			for _, field := range requiredFields {
				if _, exists := serviceConfig[field]; exists {
					fieldsFound++
				}
			}
			
			score += (float64(fieldsFound) / float64(len(requiredFields))) * 2.0
			
			// Bonus points for advanced v2.0 features
			advancedFields := []string{"dependencies", "resources", "content_types", "cli_commands"}
			for _, field := range advancedFields {
				if _, exists := serviceConfig[field]; exists {
					score += 0.25
				}
			}
			
			log.Printf("  service.json: %d/%d required fields, valid JSON", fieldsFound, len(requiredFields))
		} else {
			log.Printf("  service.json: invalid JSON - %v", err)
		}
	} else {
		log.Printf("  service.json: missing")
	}
	
	return score
}

// Validate README.md content completeness
func validateReadmeContent(readmePath string) float64 {
	score := 0.0
	
	if content, err := os.ReadFile(readmePath); err == nil {
		score += 0.5 // File exists
		
		readmeContent := strings.ToLower(string(content))
		
		// Required sections for v2.0 compliance
		requiredSections := []string{
			"installation", "configuration", "usage", "environment", 
			"troubleshooting", "api", "cli", "health",
		}
		
		sectionsFound := 0
		for _, section := range requiredSections {
			if strings.Contains(readmeContent, section) {
				sectionsFound++
			}
		}
		
		score += (float64(sectionsFound) / float64(len(requiredSections))) * 1.5
		
		// Bonus for examples and code blocks
		if strings.Contains(readmeContent, "```") || strings.Contains(readmeContent, "example") {
			score += 0.5
		}
		
		log.Printf("  README.md: %d/%d required sections found", sectionsFound, len(requiredSections))
	} else {
		log.Printf("  README.md: missing")
	}
	
	return score
}

// Enhanced health check validation
func validateHealthImplementation(resourcePath, libPath string) float64 {
	score := 0.0
	
	healthScripts := []string{
		filepath.Join(libPath, "health.sh"),
		filepath.Join(resourcePath, "health.sh"),
	}
	
	for _, healthScript := range healthScripts {
		if content, err := os.ReadFile(healthScript); err == nil {
			score += 1.0 // Health script exists
			
			scriptContent := string(content)
			
			// Check for essential health check patterns
			healthPatterns := []string{
				"timeout", "retry", "curl", "nc ", "ping", 
				"check_health", "health_status", "service_status",
			}
			
			patternsFound := 0
			for _, pattern := range healthPatterns {
				if strings.Contains(strings.ToLower(scriptContent), pattern) {
					patternsFound++
				}
			}
			
			if patternsFound >= 3 {
				score += 1.0 // Good health implementation
			}
			
			log.Printf("  health.sh: found %d/%d health patterns", patternsFound, len(healthPatterns))
			break
		}
	}
	
	return score
}

// Validate lifecycle hooks implementation
func validateLifecycleHooks(libPath string) float64 {
	score := 0.0
	
	lifecyclePath := filepath.Join(libPath, "lifecycle.sh")
	if content, err := os.ReadFile(lifecyclePath); err == nil {
		score += 1.0 // File exists
		
		scriptContent := string(content)
		
		// Required lifecycle hooks
		requiredHooks := []string{
			"start_service", "stop_service", "restart_service", 
			"service_status", "pre_start", "post_stop",
		}
		
		hooksFound := 0
		for _, hook := range requiredHooks {
			patterns := []string{
				fmt.Sprintf(`%s\s*\(\s*\)\s*\{`, hook),
				fmt.Sprintf(`function\s+%s`, hook),
			}
			
			for _, pattern := range patterns {
				if matched, _ := regexp.MatchString(pattern, scriptContent); matched {
					hooksFound++
					break
				}
			}
		}
		
		score += (float64(hooksFound) / float64(len(requiredHooks))) * 2.0
		
		log.Printf("  lifecycle.sh: %d/%d required hooks found", hooksFound, len(requiredHooks))
	} else {
		log.Printf("  lifecycle.sh: missing")
	}
	
	return score
}

// Helper functions for enhanced health checking

// Load service configuration from service.json
func loadServiceConfig(serviceJsonPath string) map[string]interface{} {
	var config map[string]interface{}
	
	if content, err := os.ReadFile(serviceJsonPath); err == nil {
		if err := json.Unmarshal(content, &config); err == nil {
			return config
		}
	}
	
	return make(map[string]interface{})
}

// Extract ports from service configuration
func getResourcePorts(serviceConfig map[string]interface{}) []int {
	var ports []int
	
	if portsConfig, exists := serviceConfig["ports"]; exists {
		switch v := portsConfig.(type) {
		case map[string]interface{}:
			for _, portValue := range v {
				if port, ok := portValue.(float64); ok {
					ports = append(ports, int(port))
				}
			}
		case []interface{}:
			for _, portValue := range v {
				if port, ok := portValue.(float64); ok {
					ports = append(ports, int(port))
				}
			}
		}
	}
	
	return ports
}

// Get hostname from service configuration
func getResourceHostname(serviceConfig map[string]interface{}) string {
	if host, exists := serviceConfig["host"]; exists {
		if hostname, ok := host.(string); ok {
			return hostname
		}
	}
	return "localhost"
}

// Check if a port is reachable
func isPortReachable(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// Check if DNS is resolvable
func isDNSResolvable(hostname string) bool {
	_, err := net.LookupHost(hostname)
	return err == nil
}

// Check if valid network interface exists
func hasValidNetworkInterface() bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				return true
			}
		}
	}
	return false
}

// Check basic routing functionality
func checkBasicRouting() bool {
	// Try to reach a well-known DNS server
	conn, err := net.DialTimeout("udp", "8.8.8.8:53", 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// Validate HTTP health endpoint
func validateHTTPHealthEndpoint(baseURL string) bool {
	endpoints := []string{"/health", "/healthz", "/ping", "/status", "/_health"}
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	for _, endpoint := range endpoints {
		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true
		}
	}
	return false
}

// Validate HTTP status endpoint
func validateHTTPStatusEndpoint(baseURL string) bool {
	endpoints := []string{"/status", "/info", "/version", "/api/status"}
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	for _, endpoint := range endpoints {
		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true
		}
	}
	return false
}

// Validate response format (JSON, etc.)
func validateResponseFormat(baseURL string) bool {
	endpoints := []string{"/health", "/status", "/api/health"}
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	for _, endpoint := range endpoints {
		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			return true
		}
		
		// Also accept text responses for simple services
		if strings.Contains(contentType, "text/") && resp.StatusCode < 400 {
			return true
		}
	}
	return false
}

// Validate response time
func validateResponseTime(baseURL string) bool {
	start := time.Now()
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		// Try alternate endpoints
		for _, endpoint := range []string{"/ping", "/status"} {
			start = time.Now()
			resp, err = client.Get(baseURL + endpoint)
			if err == nil {
				break
			}
		}
	}
	
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	duration := time.Since(start)
	return duration < 5*time.Second
}

// Validate CLI responses for non-HTTP services
func validateCLIResponses(resourceName string) float64 {
	score := 0.0
	cliName := fmt.Sprintf("resource-%s", resourceName)
	
	// Check if CLI exists
	if _, err := exec.LookPath(cliName); err != nil {
		return 0.0
	}
	
	// Test CLI commands
	commands := []string{"status", "health", "version", "help"}
	
	for _, command := range commands {
		cmd := exec.Command(cliName, command)
		if err := cmd.Run(); err == nil {
			score += 1.0
		}
	}
	
	return score
}

// Get declared dependencies from service config
func getDeclaredDependencies(serviceConfig map[string]interface{}) []string {
	var deps []string
	
	if dependencies, exists := serviceConfig["dependencies"]; exists {
		switch v := dependencies.(type) {
		case []interface{}:
			for _, dep := range v {
				if depStr, ok := dep.(string); ok {
					deps = append(deps, depStr)
				}
			}
		case map[string]interface{}:
			for depName := range v {
				deps = append(deps, depName)
			}
		}
	}
	
	return deps
}

// Check if a dependency is available
func isDependencyAvailable(dependency string) bool {
	// Check if it's a system command
	if _, err := exec.LookPath(dependency); err == nil {
		return true
	}
	
	// Check if it's a resource CLI
	resourceCLI := fmt.Sprintf("resource-%s", dependency)
	if _, err := exec.LookPath(resourceCLI); err == nil {
		return true
	}
	
	// Check if it's a running service (basic port check)
	if isServiceRunning(dependency) {
		return true
	}
	
	return false
}

// Check if a service is running by name
func isServiceRunning(serviceName string) bool {
	cmd := exec.Command("pgrep", "-f", serviceName)
	return cmd.Run() == nil
}

// Check resource-specific dependencies
func checkResourceSpecificDependencies(resourceName string) bool {
	// Define resource-specific requirements
	resourceDeps := map[string][]string{
		"postgres":    {"psql", "pg_isready"},
		"redis":       {"redis-cli"},
		"ollama":      {"curl"},
		"qdrant":      {"curl"},
		"minio":       {"mc"},
		"vault":       {"curl"},
		"browserless": {"chromium", "google-chrome", "chromium-browser"},
		"judge0":      {"docker"},
	}
	
	if deps, exists := resourceDeps[resourceName]; exists {
		for _, dep := range deps {
			if _, err := exec.LookPath(dep); err == nil {
				return true // At least one required dependency is available
			}
		}
		return false // None of the required dependencies found
	}
	
	return true // Unknown resource, assume dependencies are satisfied
}