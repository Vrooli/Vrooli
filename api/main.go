// Vrooli Unified API - Single HTTP server for all Vrooli operations
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// === Common Types ===
type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// === Orchestrator Management ===
var (
	vrooliRoot           = getVrooliRoot()
	orchestratorProcess  *exec.Cmd
	orchestratorURL      = getOrchestratorURL()
	orchestratorHealthy  = false
	orchestratorStarting = false
)

type OrchestratorConfig struct {
	Enabled  bool   `json:"enabled"`
	URL      string `json:"url"`
	Healthy  bool   `json:"healthy"`
	StartCmd string `json:"start_cmd"`
}

func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	ex, _ := os.Executable()
	return filepath.Dir(filepath.Dir(ex))
}

func getOrchestratorURL() string {
	// First check environment variable
	if port := os.Getenv("ORCHESTRATOR_PORT"); port != "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}
	
	// Try to get from port registry
	portRegistryPath := filepath.Join(getVrooliRoot(), "scripts/resources/port_registry.sh")
	cmd := exec.Command("bash", "-c", fmt.Sprintf("source %s && ports::get_resource_port vrooli-orchestrator", portRegistryPath))
	if out, err := cmd.Output(); err == nil {
		if port := strings.TrimSpace(string(out)); port != "" {
			return fmt.Sprintf("http://localhost:%s", port)
		}
	}
	
	// Fallback to default
	return "http://localhost:9500"
}

// === Orchestrator Management Functions ===

// Start the orchestrator subprocess
func startOrchestrator() error {
	if orchestratorStarting || (orchestratorProcess != nil && orchestratorProcess.Process != nil) {
		return fmt.Errorf("orchestrator already starting/running")
	}

	orchestratorStarting = true
	defer func() { orchestratorStarting = false }()

	log.Println("🚀 Starting orchestrator subprocess...")
	
	orchestratorScript := filepath.Join(vrooliRoot, "scripts/scenarios/tools/orchestrator/enhanced_orchestrator.py")
	
	// Check if orchestrator script exists
	if _, err := os.Stat(orchestratorScript); os.IsNotExist(err) {
		return fmt.Errorf("orchestrator script not found: %s", orchestratorScript)
	}

	// Start orchestrator with proper arguments
	orchestratorProcess = exec.Command("python3", orchestratorScript, "--verbose", "--fast")
	orchestratorProcess.Dir = vrooliRoot
	orchestratorProcess.Env = os.Environ()

	// Start process
	if err := orchestratorProcess.Start(); err != nil {
		orchestratorProcess = nil
		return fmt.Errorf("failed to start orchestrator: %v", err)
	}

	log.Printf("✓ Orchestrator started with PID %d", orchestratorProcess.Process.Pid)

	// Wait for orchestrator to become healthy
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		if checkOrchestratorHealth() {
			orchestratorHealthy = true
			log.Println("✓ Orchestrator is healthy and ready")
			
			// Start health monitoring in background
			go monitorOrchestratorHealth()
			return nil
		}
	}

	// If we get here, orchestrator failed to become healthy
	stopOrchestrator()
	return fmt.Errorf("orchestrator failed to become healthy after 10 seconds")
}

// Check if orchestrator is healthy
func checkOrchestratorHealth() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(orchestratorURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false
	}
	
	var healthResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return false
	}
	
	status, ok := healthResp["status"].(string)
	return ok && status == "healthy"
}

// Stop orchestrator gracefully
func stopOrchestrator() error {
	if orchestratorProcess == nil || orchestratorProcess.Process == nil {
		return nil
	}

	log.Println("🛑 Stopping orchestrator...")
	
	// Send SIGTERM for graceful shutdown
	if err := orchestratorProcess.Process.Signal(os.Interrupt); err != nil {
		log.Printf("⚠️  Failed to send interrupt signal: %v", err)
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- orchestratorProcess.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("⚠️  Orchestrator exited with error: %v", err)
		} else {
			log.Println("✓ Orchestrator stopped gracefully")
		}
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown times out
		log.Println("⚠️  Force killing orchestrator after timeout")
		orchestratorProcess.Process.Kill()
		orchestratorProcess.Wait()
	}

	orchestratorProcess = nil
	orchestratorHealthy = false
	return nil
}

// Monitor orchestrator health in background
func monitorOrchestratorHealth() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if orchestratorProcess == nil {
			return // Stop monitoring if process is nil
		}

		healthy := checkOrchestratorHealth()
		if healthy != orchestratorHealthy {
			orchestratorHealthy = healthy
			if healthy {
				log.Println("✓ Orchestrator health restored")
			} else {
				log.Println("⚠️  Orchestrator health check failed")
				
				// Restart orchestrator if it's unhealthy
				go func() {
					stopOrchestrator()
					time.Sleep(2 * time.Second)
					if err := startOrchestrator(); err != nil {
						log.Printf("❌ Failed to restart orchestrator: %v", err)
					}
				}()
				return
			}
		}
	}
}

// HTTP client for orchestrator requests
func proxyToOrchestrator(method, path string, body io.Reader) (*http.Response, error) {
	if !orchestratorHealthy {
		return nil, fmt.Errorf("orchestrator not running or unhealthy - start it with 'python3 %s/scripts/scenarios/tools/orchestrator/enhanced_orchestrator.py'", vrooliRoot)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(method, orchestratorURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

// === App Management ===
type App struct {
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Protected     bool      `json:"protected"`
	HasGit        bool      `json:"has_git"`
	Customized    bool      `json:"customized"`
	Modified      time.Time `json:"modified"`
	RuntimeStatus string    `json:"runtime_status,omitempty"` // From orchestrator
	Ports         map[string]interface{} `json:"ports,omitempty"`   // From orchestrator
	PID           *int      `json:"pid,omitempty"`             // From orchestrator
}

var appsDir = getAppsDir()

func getAppsDir() string {
	// Now using scenarios directory directly
	return filepath.Join(vrooliRoot, "scenarios")
}

func isCustomized(path string) bool {
	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		return false
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	out, _ := cmd.Output()
	if len(out) > 0 {
		return true
	}
	cmd = exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = path
	out, _ = cmd.Output()
	count := strings.TrimSpace(string(out))
	return count != "0" && count != "1"
}

func isProtected(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".vrooli", ".protected"))
	return err == nil
}

func hasGit(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// App handlers
func listApps(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Cannot read apps directory"})
		return
	}

	apps := []App{}
	
	// Get runtime status from orchestrator if available
	var orchestratorApps map[string]interface{}
	if orchestratorHealthy {
		resp, err := proxyToOrchestrator("GET", "/apps", nil)
		if err != nil {
			log.Printf("Warning: Could not get app runtime status from orchestrator: %v", err)
		} else {
			defer resp.Body.Close()
			var orchestratorData map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&orchestratorData) == nil {
				if appsList, ok := orchestratorData["apps"].([]interface{}); ok {
					orchestratorApps = make(map[string]interface{})
					for _, appData := range appsList {
						if appMap, ok := appData.(map[string]interface{}); ok {
							if name, ok := appMap["name"].(string); ok {
								orchestratorApps[name] = appMap
							}
						}
					}
				}
			}
		}
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".backups" {
			continue
		}
		appPath := filepath.Join(appsDir, entry.Name())
		info, _ := entry.Info()
		
		app := App{
			Name:       entry.Name(),
			Path:       appPath,
			Protected:  isProtected(appPath),
			HasGit:     hasGit(appPath),
			Customized: isCustomized(appPath),
			Modified:   info.ModTime(),
		}

		// Enhance with orchestrator data if available
		if orchestratorApps != nil {
			if orchData, ok := orchestratorApps[entry.Name()].(map[string]interface{}); ok {
				if status, ok := orchData["status"].(string); ok {
					app.RuntimeStatus = status
				}
				if ports, ok := orchData["allocated_ports"].(map[string]interface{}); ok {
					app.Ports = ports
				}
				if pid, ok := orchData["pid"]; ok && pid != nil {
					if pidFloat, ok := pid.(float64); ok {
						pidInt := int(pidFloat)
						app.PID = &pidInt
					}
				}
			}
		} else {
			// Fallback: basic status without orchestrator
			app.RuntimeStatus = "orchestrator_offline"
		}

		apps = append(apps, app)
	}
	json.NewEncoder(w).Encode(Response{Success: true, Data: apps})
}

func protectApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	appPath := filepath.Join(appsDir, name)
	
	if _, err := os.Stat(appPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "App not found"})
		return
	}
	
	protectDir := filepath.Join(appPath, ".vrooli")
	os.MkdirAll(protectDir, 0755)
	protectFile := filepath.Join(protectDir, ".protected")
	content := fmt.Sprintf("Protected on %s\n", time.Now().UTC().Format(time.RFC3339))
	os.WriteFile(protectFile, []byte(content), 0644)
	
	json.NewEncoder(w).Encode(Response{Success: true})
}

// Start an app via orchestrator or fallback to process manager
func startApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	
	// Validate scenario exists
	if _, err := os.Stat(scenarioPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}
	
	// Use lifecycle develop event directly - much simpler and more reliable
	// The develop phase includes auto-setup, so no need for separate setup step
	startCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s && %s/scripts/lib/utils/lifecycle.sh %s develop",
			scenarioPath, vrooliRoot, name))
	
	output, err := startCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to start scenario %s: %s", name, string(output)),
		})
		return
	}
	
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("Scenario %s started successfully", name),
			"output": string(output),
		},
	})
}

// Stop an app via orchestrator or fallback to process manager
func stopApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	
	// Use lifecycle stop event directly - much simpler and more reliable
	stopCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s/scenarios/%s && %s/scripts/lib/utils/lifecycle.sh %s stop",
			vrooliRoot, name, vrooliRoot, name))
	
	output, err := stopCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to stop app %s: %s", name, string(output)),
		})
		return
	}
	
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("App %s stopped successfully", name),
			"output": string(output),
		},
	})
}

// Restart an app
func restartApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	
	// Validate scenario exists
	if _, err := os.Stat(scenarioPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}
	
	// Stop first using lifecycle stop
	stopCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s && %s/scripts/lib/utils/lifecycle.sh %s stop",
			scenarioPath, vrooliRoot, name))
	
	stopOutput, _ := stopCmd.CombinedOutput() // Ignore stop errors - might not be running
	
	// Brief pause to ensure processes are fully stopped
	time.Sleep(2 * time.Second)
	
	// Start using lifecycle develop
	startCmd := exec.Command("bash", "-c",
		fmt.Sprintf("cd %s && %s/scripts/lib/utils/lifecycle.sh %s develop",
			scenarioPath, vrooliRoot, name))
	
	startOutput, err := startCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to restart scenario %s: %s", name, string(startOutput)),
		})
		return
	}
	
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": fmt.Sprintf("Scenario %s restarted successfully", name),
			"stop_output": string(stopOutput),
			"start_output": string(startOutput),
		},
	})
}

// Get app logs
func getAppLogs(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "50"
	}
	
	// Get logs using process manager
	logsCmd := exec.Command("bash", "-c",
		fmt.Sprintf("source %s/scripts/lib/process-manager.sh && pm::logs 'vrooli.develop.%s' %s",
			vrooliRoot, name, lines))
	
	output, err := logsCmd.CombinedOutput()
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Error: fmt.Sprintf("Failed to get logs: %s", string(output)),
		})
		return
	}
	
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"logs": string(output),
		},
	})
}

// === Scenario Management ===
type Scenario struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Template    bool   `json:"template"`
}

type CatalogEntry struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Template    bool   `json:"template,omitempty"`
}

type Catalog struct {
	Scenarios []CatalogEntry `json:"scenarios"`
}

var (
	catalogPath     = filepath.Join(vrooliRoot, "scripts/scenarios/catalog.json")
	// converterCmd removed - scenarios run directly now
)

func loadCatalog() (*Catalog, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func saveCatalog(catalog *Catalog) error {
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(catalogPath, data, 0644)
}

// Scenario handlers
func listScenarios(w http.ResponseWriter, r *http.Request) {
	catalog, err := loadCatalog()
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to load catalog"})
		return
	}

	scenarios := []Scenario{}
	for _, entry := range catalog.Scenarios {
		scenarios = append(scenarios, Scenario{
			Name:        entry.Name,
			Location:    entry.Location,
			Category:    entry.Category,
			Description: entry.Description,
			Enabled:     entry.Enabled,
			Template:    entry.Template,
		})
	}
	json.NewEncoder(w).Encode(Response{Success: true, Data: scenarios})
}

func validateScenario(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	catalog, err := loadCatalog()
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to load catalog"})
		return
	}

	var location string
	for _, entry := range catalog.Scenarios {
		if entry.Name == name {
			location = entry.Location
			break
		}
	}

	if location == "" {
		json.NewEncoder(w).Encode(Response{Error: "Scenario not found"})
		return
	}

	scenarioPath := filepath.Join(vrooliRoot, location)
	issues := []string{}
	
	serviceFile := filepath.Join(scenarioPath, "service.json")
	if _, err := os.Stat(serviceFile); err != nil {
		issues = append(issues, "Missing service.json")
	} else if data, err := os.ReadFile(serviceFile); err == nil {
		var service map[string]interface{}
		if err := json.Unmarshal(data, &service); err != nil {
			issues = append(issues, fmt.Sprintf("Invalid JSON: %v", err))
		}
	}

	result := map[string]interface{}{
		"valid":  len(issues) == 0,
		"issues": issues,
	}
	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

func getScenarioStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	
	// Check if scenario exists
	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	if _, err := os.Stat(scenarioPath); err == nil {
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data: map[string]interface{}{
				"exists": true,
				"message": "Scenario exists and ready to run",
				"path": scenarioPath,
			},
		})
	} else {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error: "Scenario not found",
		})
	}
}

// === Resource Management (placeholder) ===
func listResources(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement resource listing
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message": "Resource API coming soon",
		},
	})
}

// === Lifecycle Management (placeholder) ===
func handleLifecycle(w http.ResponseWriter, r *http.Request) {
	action := mux.Vars(r)["action"]
	// TODO: Implement lifecycle actions
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"action":  action,
			"message": "Lifecycle API coming soon",
		},
	})
}

// === Health Check ===
func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"version":     "1.0.0",
		"vrooli_root": vrooliRoot,
		"apps_dir":    appsDir,
		"apis": map[string]bool{
			"apps":      true,
			"scenarios": true,
			"resources": false, // Coming soon
			"lifecycle": false, // Coming soon
		},
	})
}

// === Orchestrator-Specific Endpoints ===

// Get running apps from orchestrator
func getRunningApps(w http.ResponseWriter, r *http.Request) {
	if !orchestratorHealthy {
		json.NewEncoder(w).Encode(Response{Error: "Orchestrator not available"})
		return
	}

	resp, err := proxyToOrchestrator("GET", "/apps/running", nil)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to get running apps"})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Invalid response from orchestrator"})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

// Start all enabled apps
func startAllApps(w http.ResponseWriter, r *http.Request) {
	if !orchestratorHealthy {
		json.NewEncoder(w).Encode(Response{Error: "Orchestrator not available"})
		return
	}

	resp, err := proxyToOrchestrator("POST", "/apps/start-all", nil)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to start all apps"})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Invalid response from orchestrator"})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

// Stop all running apps
func stopAllApps(w http.ResponseWriter, r *http.Request) {
	if !orchestratorHealthy {
		json.NewEncoder(w).Encode(Response{Error: "Orchestrator not available"})
		return
	}

	resp, err := proxyToOrchestrator("POST", "/apps/stop-all", nil)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to stop all apps"})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Invalid response from orchestrator"})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

// Get detailed app status from orchestrator
func getDetailedAppStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	
	if !orchestratorHealthy {
		json.NewEncoder(w).Encode(Response{Error: "Orchestrator not available"})
		return
	}

	resp, err := proxyToOrchestrator("GET", fmt.Sprintf("/apps/%s/status", name), nil)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to get app status"})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Invalid response from orchestrator"})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: result})
}

// Get orchestrator status
func getOrchestratorStatus(w http.ResponseWriter, r *http.Request) {
	status := OrchestratorConfig{
		Enabled: true,
		URL:     orchestratorURL,
		Healthy: orchestratorHealthy,
		StartCmd: "python3 scripts/scenarios/tools/orchestrator/enhanced_orchestrator.py --verbose --fast",
	}

	if orchestratorHealthy {
		resp, err := proxyToOrchestrator("GET", "/status", nil)
		if err == nil {
			defer resp.Body.Close()
			var orchStatus map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&orchStatus) == nil {
				json.NewEncoder(w).Encode(Response{
					Success: true,
					Data: map[string]interface{}{
						"orchestrator": status,
						"system": orchStatus,
					},
				})
				return
			}
		}
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"orchestrator": status,
		},
	})
}

func main() {
	port := os.Getenv("VROOLI_API_PORT")
	if port == "" {
		port = "8092"
	}

	// Start orchestrator
	log.Println("🎛️  Starting orchestrator...")
	if err := startOrchestrator(); err != nil {
		log.Printf("⚠️  Failed to start orchestrator: %v", err)
		log.Println("   Continuing without orchestrator (fallback mode)")
	}

	// Handle shutdown signals
	go func() {
		// Wait for interrupt signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		<-ch
		
		log.Println("🛑 Shutting down gracefully...")
		if err := stopOrchestrator(); err != nil {
			log.Printf("⚠️  Error stopping orchestrator: %v", err)
		}
		os.Exit(0)
	}()

	r := mux.NewRouter()
	
	// Health
	r.HandleFunc("/health", healthCheck).Methods("GET")
	
	// Apps API - Enhanced with orchestrator
	r.HandleFunc("/apps", listApps).Methods("GET")
	r.HandleFunc("/apps/running", getRunningApps).Methods("GET")
	r.HandleFunc("/apps/start-all", startAllApps).Methods("POST")
	r.HandleFunc("/apps/stop-all", stopAllApps).Methods("POST")
	r.HandleFunc("/apps/{name}/protect", protectApp).Methods("POST")
	r.HandleFunc("/apps/{name}/start", startApp).Methods("POST")
	r.HandleFunc("/apps/{name}/stop", stopApp).Methods("POST")
	r.HandleFunc("/apps/{name}/restart", restartApp).Methods("POST")
	r.HandleFunc("/apps/{name}/logs", getAppLogs).Methods("GET")
	r.HandleFunc("/apps/{name}/status", getDetailedAppStatus).Methods("GET")
	
	// Orchestrator API
	r.HandleFunc("/orchestrator/status", getOrchestratorStatus).Methods("GET")
	
	// Scenarios API
	r.HandleFunc("/scenarios", listScenarios).Methods("GET")
	r.HandleFunc("/scenarios/{name}/validate", validateScenario).Methods("POST")
	r.HandleFunc("/scenarios/{name}/status", getScenarioStatus).Methods("GET")
	
	// Resources API (placeholder)
	r.HandleFunc("/resources", listResources).Methods("GET")
	
	// Lifecycle API (placeholder)
	r.HandleFunc("/lifecycle/{action}", handleLifecycle).Methods("POST")

	log.Printf("🚀 Vrooli Unified API running on port %s", port)
	log.Printf("   Orchestrator: %s", map[bool]string{true: "✅ healthy", false: "❌ unavailable"}[orchestratorHealthy])
	log.Fatal(http.ListenAndServe(":"+port, r))
}