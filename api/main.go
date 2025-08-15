// Vrooli Unified API - Single HTTP server for all Vrooli operations
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
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

var vrooliRoot = getVrooliRoot()

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

// === App Management ===
type App struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Protected  bool      `json:"protected"`
	HasGit     bool      `json:"has_git"`
	Customized bool      `json:"customized"`
	Modified   time.Time `json:"modified"`
}

var appsDir = getAppsDir()

func getAppsDir() string {
	if dir := os.Getenv("GENERATED_APPS_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "generated-apps")
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
	converterCmd    = filepath.Join(vrooliRoot, "scripts/scenarios/tools/scenario-to-app.sh")
	conversionJobs  = make(map[string]*ConversionStatus)
)

type ConversionStatus struct {
	InProgress bool   `json:"in_progress"`
	Message    string `json:"message"`
	StartTime  string `json:"start_time,omitempty"`
}

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

func convertScenario(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	
	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"
	
	if status, exists := conversionJobs[name]; exists && status.InProgress {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Conversion already in progress",
			Data:    status,
		})
		return
	}

	status := &ConversionStatus{
		InProgress: true,
		Message:    "Starting conversion...",
		StartTime:  time.Now().Format(time.RFC3339),
	}
	conversionJobs[name] = status

	go func() {
		var cmd *exec.Cmd
		if force {
			cmd = exec.Command(converterCmd, name, "--force")
		} else {
			cmd = exec.Command(converterCmd, name)
		}
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			status.InProgress = false
			status.Message = fmt.Sprintf("Failed: %s", string(output))
		} else {
			status.InProgress = false
			status.Message = "Completed successfully"
		}
		
		time.Sleep(5 * time.Minute)
		delete(conversionJobs, name)
	}()

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "Conversion started",
			"status":  status,
		},
	})
}

func getScenarioStatus(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	
	if status, exists := conversionJobs[name]; exists {
		json.NewEncoder(w).Encode(Response{Success: true, Data: status})
	} else {
		// No active conversion job - check if app exists
		appPath := filepath.Join(appsDir, name)
		if _, err := os.Stat(appPath); err == nil {
			json.NewEncoder(w).Encode(Response{
				Success: true,
				Data: &ConversionStatus{
					InProgress: false,
					Message:    "App exists and ready",
				},
			})
		} else {
			json.NewEncoder(w).Encode(Response{
				Success: true,
				Data: &ConversionStatus{
					InProgress: false,
					Message:    "No conversion in progress",
				},
			})
		}
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

func main() {
	port := os.Getenv("VROOLI_API_PORT")
	if port == "" {
		port = "8090"
	}

	r := mux.NewRouter()
	
	// Health
	r.HandleFunc("/health", healthCheck).Methods("GET")
	
	// Apps API
	r.HandleFunc("/apps", listApps).Methods("GET")
	r.HandleFunc("/apps/{name}/protect", protectApp).Methods("POST")
	
	// Scenarios API
	r.HandleFunc("/scenarios", listScenarios).Methods("GET")
	r.HandleFunc("/scenarios/{name}/validate", validateScenario).Methods("POST")
	r.HandleFunc("/scenarios/{name}/convert", convertScenario).Methods("POST")
	r.HandleFunc("/scenarios/{name}/status", getScenarioStatus).Methods("GET")
	
	// Resources API (placeholder)
	r.HandleFunc("/resources", listResources).Methods("GET")
	
	// Lifecycle API (placeholder)
	r.HandleFunc("/lifecycle/{action}", handleLifecycle).Methods("POST")

	log.Printf("🚀 Vrooli Unified API running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}