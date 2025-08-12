// Vrooli App Management API - Minimal HTTP server for app operations
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

type App struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Protected  bool      `json:"protected"`
	HasGit     bool      `json:"has_git"`
	Customized bool      `json:"customized"`
	Modified   time.Time `json:"modified"`
}

type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

var appsDir = getAppsDir()

func getAppsDir() string {
	if dir := os.Getenv("GENERATED_APPS_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "generated-apps")
}

// Check if app has uncommitted changes or additional commits
func isCustomized(path string) bool {
	// Check for git repo
	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		return false
	}

	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	out, _ := cmd.Output()
	if len(out) > 0 {
		return true
	}

	// Check commit count (> 1 means customized)
	cmd = exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = path
	out, _ = cmd.Output()
	count := strings.TrimSpace(string(out))
	return count != "0" && count != "1"
}

// List all apps with status
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

// Get specific app status
func getApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	appPath := filepath.Join(appsDir, name)

	info, err := os.Stat(appPath)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Error: "App not found"})
		return
	}

	app := App{
		Name:       name,
		Path:       appPath,
		Protected:  isProtected(appPath),
		HasGit:     hasGit(appPath),
		Customized: isCustomized(appPath),
		Modified:   info.ModTime(),
	}

	// Get backup count
	backups := 0
	backupPattern := filepath.Join(appsDir, ".backups", name+"-*")
	matches, _ := filepath.Glob(backupPattern)
	backups = len(matches)

	data := map[string]interface{}{
		"app":     app,
		"backups": backups,
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

// Protect app from regeneration
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

// Remove protection
func unprotectApp(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	protectFile := filepath.Join(appsDir, name, ".vrooli", ".protected")
	os.Remove(protectFile)
	json.NewEncoder(w).Encode(Response{Success: true})
}

// Create backup
func createBackup(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	appPath := filepath.Join(appsDir, name)

	if _, err := os.Stat(appPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "App not found"})
		return
	}

	backupDir := filepath.Join(appsDir, ".backups")
	os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s-%s", name, timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	cmd := exec.Command("cp", "-r", appPath, backupPath)
	if err := cmd.Run(); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to create backup"})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    map[string]string{"backup": backupName, "path": backupPath},
	})
}

// Restore from backup
func restoreBackup(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	
	var req struct {
		Backup string `json:"backup"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Handle "latest"
	backupName := req.Backup
	if backupName == "latest" {
		pattern := filepath.Join(appsDir, ".backups", name+"-*")
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			json.NewEncoder(w).Encode(Response{Error: "No backups found"})
			return
		}
		backupName = filepath.Base(matches[len(matches)-1])
	}

	backupPath := filepath.Join(appsDir, ".backups", backupName)
	if _, err := os.Stat(backupPath); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Backup not found"})
		return
	}

	appPath := filepath.Join(appsDir, name)
	os.RemoveAll(appPath)
	
	cmd := exec.Command("cp", "-r", backupPath, appPath)
	if err := cmd.Run(); err != nil {
		json.NewEncoder(w).Encode(Response{Error: "Failed to restore"})
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, Data: "Restored from " + backupName})
}

// Helper functions
func isProtected(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".vrooli", ".protected"))
	return err == nil
}

func hasGit(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"app_dir": appsDir,
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8094"
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", healthCheck).Methods("GET")
	r.HandleFunc("/apps", listApps).Methods("GET")
	r.HandleFunc("/apps/{name}", getApp).Methods("GET")
	r.HandleFunc("/apps/{name}/protect", protectApp).Methods("POST")
	r.HandleFunc("/apps/{name}/protect", unprotectApp).Methods("DELETE")
	r.HandleFunc("/apps/{name}/backup", createBackup).Methods("POST")
	r.HandleFunc("/apps/{name}/restore", restoreBackup).Methods("POST")

	log.Printf("📱 Vrooli App API running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}