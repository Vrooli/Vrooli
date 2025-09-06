package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Project represents a project in the system
type Project struct {
	ID               string            `json:"id"`
	Path             string            `json:"path"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	VrooliVersion    *string           `json:"vrooli_version"`
	BridgeVersion    *string           `json:"bridge_version"`
	IntegrationStatus string           `json:"integration_status"`
	LastUpdated      time.Time         `json:"last_updated"`
	CreatedAt        time.Time         `json:"created_at"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ScanRequest represents a request to scan for projects
type ScanRequest struct {
	Directories []string `json:"directories"`
	Depth       int      `json:"depth"`
}

// ScanResponse represents the response from a scan operation
type ScanResponse struct {
	Found    int       `json:"found"`
	New      int       `json:"new"`
	Projects []Project `json:"projects"`
}

// IntegrateRequest represents a request to integrate a project
type IntegrateRequest struct {
	Force bool `json:"force"`
}

// IntegrateResponse represents the response from an integration operation
type IntegrateResponse struct {
	Success       bool     `json:"success"`
	FilesCreated  []string `json:"files_created"`
	FilesUpdated  []string `json:"files_updated"`
	Message       string   `json:"message"`
}

// ProjectType represents information about a project type
type ProjectType struct {
	Name                 string   `json:"name"`
	Detection            []string `json:"detection"`
	PreferredScenarios   []string `json:"preferred_scenarios"`
	SpecificCapabilities string   `json:"specific_capabilities"`
	Commands             []string `json:"commands"`
}

// App represents the main application
type App struct {
	db           *sql.DB
	projectTypes map[string]ProjectType
}

func main() {
	app := &App{}
	
	// Initialize database connection
	if err := app.initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer app.db.Close()
	
	// Load project type definitions
	if err := app.loadProjectTypes(); err != nil {
		log.Fatal("Failed to load project types:", err)
	}
	
	// Setup routes
	router := app.setupRoutes()
	
	// Get port from environment or use default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Starting Vrooli Bridge API on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func (app *App) initDB() error {
	// Get database connection details from environment
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}
	
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "postgres"
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	
	var err error
	app.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	
	return app.db.Ping()
}

func (app *App) loadProjectTypes() error {
	// Load project types from the templates directory
	data, err := ioutil.ReadFile("../initialization/templates/project-types.json")
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &app.projectTypes)
}

func (app *App) setupRoutes() *mux.Router {
	r := mux.NewRouter()
	
	// Enable CORS
	r.Use(app.corsMiddleware)
	
	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/projects", app.getProjects).Methods("GET")
	api.HandleFunc("/projects/scan", app.scanProjects).Methods("POST")
	api.HandleFunc("/projects/{id}/integrate", app.integrateProject).Methods("POST")
	api.HandleFunc("/projects/{id}", app.deleteProject).Methods("DELETE")
	
	// Health check
	r.HandleFunc("/health", app.healthCheck).Methods("GET")
	
	return r
}

func (app *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (app *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check database connection
	if err := app.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"version": "1.0.0",
	})
}

func (app *App) getProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	rows, err := app.db.Query(`
		SELECT id, path, name, type, vrooli_version, bridge_version, 
		       integration_status, last_updated, created_at, metadata
		FROM projects
		ORDER BY last_updated DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var projects []Project
	for rows.Next() {
		var p Project
		var metadataJSON []byte
		
		err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.Type, &p.VrooliVersion, 
			&p.BridgeVersion, &p.IntegrationStatus, &p.LastUpdated, &p.CreatedAt, &metadataJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &p.Metadata)
		}
		
		projects = append(projects, p)
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"projects": projects,
	})
}

func (app *App) scanProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Default values
	if len(req.Directories) == 0 {
		homeDir, _ := os.UserHomeDir()
		req.Directories = []string{homeDir}
	}
	if req.Depth <= 0 {
		req.Depth = 3
	}
	
	var allProjects []Project
	var newCount int
	
	for _, dir := range req.Directories {
		projects := app.scanDirectory(dir, req.Depth)
		for _, project := range projects {
			// Check if project already exists
			exists, err := app.projectExists(project.Path)
			if err != nil {
				log.Printf("Error checking project existence: %v", err)
				continue
			}
			
			if !exists {
				// Insert new project
				err := app.insertProject(project)
				if err != nil {
					log.Printf("Error inserting project: %v", err)
					continue
				}
				newCount++
			}
			
			allProjects = append(allProjects, project)
		}
	}
	
	response := ScanResponse{
		Found:    len(allProjects),
		New:      newCount,
		Projects: allProjects,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (app *App) scanDirectory(rootDir string, maxDepth int) []Project {
	var projects []Project
	
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue scanning despite errors
		}
		
		// Check depth
		relPath, _ := filepath.Rel(rootDir, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Skip hidden directories and common non-project directories
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || 
			info.Name() == "node_modules" || info.Name() == "target" || 
			info.Name() == "__pycache__") {
			return filepath.SkipDir
		}
		
		if !info.IsDir() {
			// Check if this file indicates a project
			projectType := app.detectProjectType(path, info.Name())
			if projectType != "" {
				projectDir := filepath.Dir(path)
				projectName := filepath.Base(projectDir)
				
				project := Project{
					ID:               uuid.New().String(),
					Path:             projectDir,
					Name:             projectName,
					Type:             projectType,
					IntegrationStatus: "missing",
					CreatedAt:        time.Now(),
					LastUpdated:      time.Now(),
					Metadata:         make(map[string]interface{}),
				}
				
				projects = append(projects, project)
			}
		}
		
		return nil
	})
	
	return projects
}

func (app *App) detectProjectType(filePath, fileName string) string {
	for typeName, typeInfo := range app.projectTypes {
		for _, pattern := range typeInfo.Detection {
			if matched, _ := filepath.Match(pattern, fileName); matched {
				return typeName
			}
		}
	}
	return ""
}

func (app *App) projectExists(path string) (bool, error) {
	var count int
	err := app.db.QueryRow("SELECT COUNT(*) FROM projects WHERE path = $1", path).Scan(&count)
	return count > 0, err
}

func (app *App) insertProject(project Project) error {
	metadataJSON, _ := json.Marshal(project.Metadata)
	
	_, err := app.db.Exec(`
		INSERT INTO projects (id, path, name, type, integration_status, created_at, last_updated, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, project.ID, project.Path, project.Name, project.Type, project.IntegrationStatus,
		project.CreatedAt, project.LastUpdated, metadataJSON)
	
	return err
}

func (app *App) integrateProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	projectID := vars["id"]
	
	var req IntegrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to non-force if request body is empty
		req.Force = false
	}
	
	// Get project from database
	project, err := app.getProjectByID(projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	
	// Perform integration
	filesCreated, filesUpdated, err := app.performIntegration(project, req.Force)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update project status
	_, err = app.db.Exec(`
		UPDATE projects 
		SET integration_status = 'active', 
		    vrooli_version = $1, 
		    bridge_version = $2,
		    last_updated = NOW()
		WHERE id = $3
	`, "1.0.0", "1.0.0", projectID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := IntegrateResponse{
		Success:      true,
		FilesCreated: filesCreated,
		FilesUpdated: filesUpdated,
		Message:      "Integration completed successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

func (app *App) getProjectByID(id string) (*Project, error) {
	var p Project
	var metadataJSON []byte
	
	err := app.db.QueryRow(`
		SELECT id, path, name, type, vrooli_version, bridge_version,
		       integration_status, last_updated, created_at, metadata
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.Path, &p.Name, &p.Type, &p.VrooliVersion,
		&p.BridgeVersion, &p.IntegrationStatus, &p.LastUpdated, &p.CreatedAt, &metadataJSON)
	
	if err != nil {
		return nil, err
	}
	
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &p.Metadata)
	}
	
	return &p, nil
}

func (app *App) performIntegration(project *Project, force bool) ([]string, []string, error) {
	var filesCreated []string
	var filesUpdated []string
	
	// Generate Vrooli integration documentation
	integrationFile := filepath.Join(project.Path, "VROOLI_INTEGRATION.md")
	
	if !app.fileExists(integrationFile) || force {
		content := app.generateIntegrationDoc(project)
		err := ioutil.WriteFile(integrationFile, []byte(content), 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to write integration file: %v", err)
		}
		
		if app.fileExists(integrationFile) && !force {
			filesUpdated = append(filesUpdated, integrationFile)
		} else {
			filesCreated = append(filesCreated, integrationFile)
		}
	}
	
	// Update or create CLAUDE.md additions
	claudeFile := filepath.Join(project.Path, "CLAUDE.md")
	claudeAddition := app.generateClaudeAddition(project)
	
	if app.fileExists(claudeFile) {
		// Append to existing CLAUDE.md
		content, err := ioutil.ReadFile(claudeFile)
		if err == nil && !strings.Contains(string(content), "Vrooli Integration") {
			newContent := string(content) + "\n\n" + claudeAddition
			err = ioutil.WriteFile(claudeFile, []byte(newContent), 0644)
			if err != nil {
				return filesCreated, filesUpdated, fmt.Errorf("failed to update CLAUDE.md: %v", err)
			}
			filesUpdated = append(filesUpdated, claudeFile)
		}
	} else {
		// Create new CLAUDE.md
		err := ioutil.WriteFile(claudeFile, []byte(claudeAddition), 0644)
		if err != nil {
			return filesCreated, filesUpdated, fmt.Errorf("failed to create CLAUDE.md: %v", err)
		}
		filesCreated = append(filesCreated, claudeFile)
	}
	
	return filesCreated, filesUpdated, nil
}

func (app *App) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func (app *App) generateIntegrationDoc(project *Project) string {
	// Load template
	template, err := ioutil.ReadFile("../initialization/templates/VROOLI_INTEGRATION.md.template")
	if err != nil {
		// Return basic template if file not found
		return fmt.Sprintf("# Vrooli Integration\n\nThis project has been integrated with Vrooli.\n\nProject: %s\nType: %s\n", project.Name, project.Type)
	}
	
	content := string(template)
	
	// Get project type info
	projectTypeInfo, exists := app.projectTypes[project.Type]
	if !exists {
		projectTypeInfo = app.projectTypes["generic"]
	}
	
	// Replace template variables
	replacements := map[string]string{
		"{{date}}":                    time.Now().Format("2006-01-02"),
		"{{vrooli_version}}":          "1.0.0",
		"{{project_type}}":            projectTypeInfo.Name,
		"{{project_name}}":            project.Name,
		"{{project_path}}":            project.Path,
		"{{bridge_version}}":          "1.0.0",
		"{{project_type_specific}}":   projectTypeInfo.SpecificCapabilities,
		"{{preferred_scenarios}}":     strings.Join(projectTypeInfo.PreferredScenarios, ", "),
	}
	
	for placeholder, value := range replacements {
		content = strings.ReplaceAll(content, placeholder, value)
	}
	
	return content
}

func (app *App) generateClaudeAddition(project *Project) string {
	// Load template
	template, err := ioutil.ReadFile("../initialization/templates/CLAUDE_ADDITIONS.md.template")
	if err != nil {
		// Return basic template if file not found
		return fmt.Sprintf("# Vrooli Integration\n\nThis project has Vrooli integration. See VROOLI_INTEGRATION.md for details.\n")
	}
	
	content := string(template)
	
	// Get project type info
	projectTypeInfo, exists := app.projectTypes[project.Type]
	if !exists {
		projectTypeInfo = app.projectTypes["generic"]
	}
	
	// Replace template variables
	replacements := map[string]string{
		"{{date}}":                         time.Now().Format("2006-01-02"),
		"{{vrooli_version}}":               "1.0.0",
		"{{project_type}}":                 projectTypeInfo.Name,
		"{{project_name}}":                 project.Name,
		"{{project_path}}":                 project.Path,
		"{{bridge_version}}":               "1.0.0",
		"{{project_specific_capabilities}}": projectTypeInfo.SpecificCapabilities,
	}
	
	for placeholder, value := range replacements {
		content = strings.ReplaceAll(content, placeholder, value)
	}
	
	return content
}

func (app *App) deleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]
	
	// Get project info first
	project, err := app.getProjectByID(projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	
	// Remove integration files
	integrationFile := filepath.Join(project.Path, "VROOLI_INTEGRATION.md")
	if app.fileExists(integrationFile) {
		os.Remove(integrationFile)
	}
	
	// Remove from CLAUDE.md (more complex - would need to parse and remove section)
	// For now, we'll just mark as removed in database
	
	// Update project status to missing
	_, err = app.db.Exec(`
		UPDATE projects 
		SET integration_status = 'missing',
		    last_updated = NOW()
		WHERE id = $1
	`, projectID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Integration removed successfully",
	})
}