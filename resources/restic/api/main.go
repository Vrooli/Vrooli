package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "time"
)

type HealthResponse struct {
    Status     string    `json:"status"`
    Repository string    `json:"repository"`
    Timestamp  time.Time `json:"timestamp"`
}

type BackupRequest struct {
    Paths []string `json:"paths"`
    Tags  []string `json:"tags,omitempty"`
}

type BackupResponse struct {
    Success  bool   `json:"success"`
    Snapshot string `json:"snapshot,omitempty"`
    Error    string `json:"error,omitempty"`
}

type SnapshotInfo struct {
    ID       string    `json:"id"`
    Time     time.Time `json:"time"`
    Hostname string    `json:"hostname"`
    Tags     []string  `json:"tags"`
    Paths    []string  `json:"paths"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:     "healthy",
        Repository: os.Getenv("RESTIC_REPOSITORY"),
        Timestamp:  time.Now(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req BackupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Build restic backup command
    args := []string{"backup"}
    args = append(args, req.Paths...)
    
    for _, tag := range req.Tags {
        args = append(args, "--tag", tag)
    }
    
    cmd := exec.Command("restic", args...)
    cmd.Env = os.Environ()
    
    output, err := cmd.CombinedOutput()
    
    response := BackupResponse{}
    if err != nil {
        response.Success = false
        response.Error = string(output)
    } else {
        response.Success = true
        response.Snapshot = "backup-completed"
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func snapshotsHandler(w http.ResponseWriter, r *http.Request) {
    cmd := exec.Command("restic", "snapshots", "--json")
    cmd.Env = os.Environ()
    
    output, err := cmd.Output()
    if err != nil {
        http.Error(w, "Failed to list snapshots", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.Write(output)
}

func main() {
    port := os.Getenv("API_PORT")
    if port == "" {
        port = "8000"
    }
    
    // Ensure password is set for encryption
    if os.Getenv("RESTIC_PASSWORD") == "" {
        log.Fatal("RESTIC_PASSWORD environment variable must be set for encryption")
    }
    
    // Initialize repository if it doesn't exist
    initCmd := exec.Command("restic", "init")
    initCmd.Env = os.Environ()
    output, err := initCmd.CombinedOutput()
    if err != nil {
        // Check if it's already initialized
        if !strings.Contains(string(output), "config file already exists") {
            log.Printf("Repository initialization warning: %s", output)
        }
    } else {
        log.Println("Repository initialized with AES-256 encryption")
    }
    
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/backup", backupHandler)
    http.HandleFunc("/snapshots", snapshotsHandler)
    
    log.Printf("Restic API server starting on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}