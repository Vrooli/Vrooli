package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

// Task represents a task in the system
type Task struct {
	ID                string                 `yaml:"id" json:"id"`
	Title             string                 `yaml:"title" json:"title"`
	Description       string                 `yaml:"description" json:"description"`
	Type              string                 `yaml:"type" json:"type"`
	Target            string                 `yaml:"target" json:"target"`
	PriorityEstimates map[string]interface{} `yaml:"priority_estimates" json:"priority_estimates"`
	PriorityScore     *float64               `yaml:"priority_score" json:"priority_score"`
	Dependencies      []string               `yaml:"dependencies" json:"dependencies"`
	Blockers          []string               `yaml:"blockers" json:"blockers"`
	CreatedAt         time.Time              `yaml:"created_at" json:"created_at"`
	CreatedBy         string                 `yaml:"created_by" json:"created_by"`
	AnalyzedAt        *time.Time             `yaml:"analyzed_at" json:"analyzed_at"`
	StartedAt         *time.Time             `yaml:"started_at" json:"started_at"`
	CompletedAt       *time.Time             `yaml:"completed_at" json:"completed_at"`
	Attempts          []interface{}          `yaml:"attempts" json:"attempts"`
	Notes             string                 `yaml:"notes" json:"notes"`
	Status            string                 `json:"status"` // Derived from folder location
}

// Agent represents an active agent
type Agent struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	CurrentTaskID   *string   `json:"current_task_id"`
	CurrentTaskTitle *string  `json:"current_task_title"`
	Status          string    `json:"status"`
	StartedAt       *time.Time `json:"started_at"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	ResourceUsage   map[string]interface{} `json:"resource_usage"`
}

// Problem represents a discovered system problem
type Problem struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Severity        string                 `json:"severity"` // critical|high|medium|low
	Frequency       string                 `json:"frequency"` // constant|frequent|occasional|rare
	Impact          string                 `json:"impact"` // system_down|degraded_performance|user_impact|cosmetic
	Status          string                 `json:"status"` // active|investigating|resolved|ignored
	DiscoveredAt    time.Time              `json:"discovered_at"`
	DiscoveredBy    string                 `json:"discovered_by"`
	LastOccurrence  *time.Time             `json:"last_occurrence"`
	SourceFile      string                 `json:"source_file"`
	AffectedComponents []string            `json:"affected_components"`
	Symptoms        []string               `json:"symptoms"`
	Evidence        map[string]interface{} `json:"evidence"`
	RelatedIssues   []string               `json:"related_issues"`
	PriorityEstimates map[string]interface{} `json:"priority_estimates"`
	Resolution      *string                `json:"resolution"`
	ResolvedAt      *time.Time             `json:"resolved_at"`
	ResolvedBy      *string                `json:"resolved_by"`
	TasksCreated    []string               `json:"tasks_created"`
}

// ProblemScanRequest represents a request to scan for problems
type ProblemScanRequest struct {
	ScanPath string `json:"scan_path"`
	Force    bool   `json:"force"`
}

// ProblemScanResponse represents the response from a problem scan
type ProblemScanResponse struct {
	ProblemsFound int      `json:"problems_found"`
	TasksCreated  int      `json:"tasks_created"`
	ScanTime      float64  `json:"scan_time"`
	NewProblems   []string `json:"new_problems"`
}

// Config represents system configuration
type Config struct {
	MaxConcurrentTasks    int     `json:"max_concurrent_tasks"`
	MinBacklogSize        int     `json:"min_backlog_size"`
	TaskScanInterval      int     `json:"task_scan_interval"`
	YoloMode              bool    `json:"yolo_mode"`
	MaxCPUPercent         int     `json:"max_cpu_percent"`
	MaxMemoryPercent      int     `json:"max_memory_percent"`
	MaxClaudeCallsPerHour int     `json:"max_claude_calls_per_hour"`
	PriorityWeights       map[string]float64 `json:"priority_weights"`
}

var (
	db       *sql.DB
	tasksDir = "/home/matthalloran8/Vrooli/scenarios/swarm-manager/tasks"
)

func main() {
	// Initialize database connection
	initDB()
	defer db.Close()

	// Start agent system
	startAgentSystem()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Routes
	setupRoutes(app)

	// Get port from environment or use default
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8095"
	}

	// Start server
	log.Printf("Swarm Manager API starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

func initDB() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	connStr := fmt.Sprintf("host=%s port=5433 user=postgres dbname=postgres sslmode=disable", dbHost)
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to PostgreSQL database")
}

func setupRoutes(app *fiber.App) {
	// Health check
	app.Get("/health", healthCheck)

	// Task endpoints
	app.Get("/api/tasks", getTasks)
	app.Post("/api/tasks", createTask)
	app.Put("/api/tasks/:id", updateTask)
	app.Delete("/api/tasks/:id", deleteTask)
	app.Post("/api/tasks/:id/execute", executeTask)
	app.Post("/api/tasks/:id/analyze", analyzeTask)

	// Agent endpoints
	app.Get("/api/agents", getAgents)
	app.Post("/api/agents/:name/heartbeat", agentHeartbeat)

	// Metrics endpoints
	app.Get("/api/metrics", getMetrics)
	app.Get("/api/metrics/success-rate", getSuccessRate)

	// Configuration endpoints
	app.Get("/api/config", getConfig)
	app.Put("/api/config", updateConfig)

	// Problem endpoints
	app.Post("/api/problems/scan", scanProblems)
	app.Get("/api/problems", getProblems)
	app.Get("/api/problems/:id", getProblem)
	app.Put("/api/problems/:id/resolve", resolveProblem)

	// Priority calculation
	app.Post("/api/calculate-priority", calculatePriority)
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"service": "swarm-manager",
		"timestamp": time.Now().Unix(),
	})
}

func getTasks(c *fiber.Ctx) error {
	status := c.Query("status", "all")
	tasks := []Task{}

	// Scan task directories based on status
	folders := map[string]string{
		"active":    filepath.Join(tasksDir, "active"),
		"staged":    filepath.Join(tasksDir, "staged"),
		"backlog":   filepath.Join(tasksDir, "backlog"),
		"completed": filepath.Join(tasksDir, "completed"),
		"failed":    filepath.Join(tasksDir, "failed"),
	}

	for folderStatus, folderPath := range folders {
		if status != "all" && status != folderStatus {
			continue
		}

		// Read tasks from folder
		folderTasks := readTasksFromFolder(folderPath, folderStatus)
		tasks = append(tasks, folderTasks...)

		// Also check subfolders for backlog
		if folderStatus == "backlog" {
			manualTasks := readTasksFromFolder(filepath.Join(folderPath, "manual"), "backlog")
			generatedTasks := readTasksFromFolder(filepath.Join(folderPath, "generated"), "backlog")
			tasks = append(tasks, manualTasks...)
			tasks = append(tasks, generatedTasks...)
		}
	}

	return c.JSON(fiber.Map{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func readTasksFromFolder(folderPath, status string) []Task {
	tasks := []Task{}
	
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		// Folder might not exist yet
		return tasks
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			continue
		}

		taskPath := filepath.Join(folderPath, file.Name())
		data, err := ioutil.ReadFile(taskPath)
		if err != nil {
			continue
		}

		var task Task
		if err := yaml.Unmarshal(data, &task); err != nil {
			continue
		}

		task.Status = status
		tasks = append(tasks, task)
	}

	return tasks
}

func createTask(c *fiber.Ctx) error {
	var task Task
	if err := c.BodyParser(&task); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Generate ID if not provided
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%s", uuid.New().String()[:8])
	}

	// Set creation time
	task.CreatedAt = time.Now()
	if task.CreatedBy == "" {
		task.CreatedBy = "api"
	}

	// Determine folder based on created_by
	folder := filepath.Join(tasksDir, "backlog", "manual")
	if task.CreatedBy == "ai" {
		folder = filepath.Join(tasksDir, "backlog", "generated")
	}

	// Create folder if it doesn't exist
	os.MkdirAll(folder, 0755)

	// Write task to file
	taskPath := filepath.Join(folder, fmt.Sprintf("%s.yaml", task.ID))
	data, err := yaml.Marshal(&task)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to serialize task",
		})
	}

	if err := ioutil.WriteFile(taskPath, data, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to write task file",
		})
	}

	// Log to database
	_, err = db.Exec(`
		INSERT INTO swarm_manager.task_executions 
		(task_id, task_title, task_type, status, created_at)
		VALUES ($1, $2, $3, 'pending', $4)`,
		task.ID, task.Title, task.Type, task.CreatedAt)
	
	if err != nil {
		log.Printf("Failed to log task creation: %v", err)
	}

	return c.Status(201).JSON(task)
}

func updateTask(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Find task file
	taskPath := findTaskFile(id)
	if taskPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	// Read existing task
	data, err := ioutil.ReadFile(taskPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to read task",
		})
	}

	var task Task
	if err := yaml.Unmarshal(data, &task); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to parse task",
		})
	}

	// Update with new data
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Apply updates (simplified - in production would be more sophisticated)
	if title, ok := updates["title"].(string); ok {
		task.Title = title
	}
	if desc, ok := updates["description"].(string); ok {
		task.Description = desc
	}
	if notes, ok := updates["notes"].(string); ok {
		task.Notes = notes
	}

	// Write back to file
	data, err = yaml.Marshal(&task)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to serialize task",
		})
	}

	if err := ioutil.WriteFile(taskPath, data, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to write task file",
		})
	}

	return c.JSON(task)
}

func deleteTask(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Find and delete task file
	taskPath := findTaskFile(id)
	if taskPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	if err := os.Remove(taskPath); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete task",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Task deleted successfully",
		"id": id,
	})
}

func executeTask(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Find task file
	taskPath := findTaskFile(id)
	if taskPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	// Move to active folder
	activePath := filepath.Join(tasksDir, "active", filepath.Base(taskPath))
	if err := os.Rename(taskPath, activePath); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to move task to active",
		})
	}

	// Execute task using the execution engine
	go executeTaskAsync(id, taskPath, activePath)

	return c.JSON(fiber.Map{
		"message": "Task execution started",
		"id": id,
	})
}

func analyzeTask(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Find task file
	taskPath := findTaskFile(id)
	if taskPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	// Analyze task using Claude (async to avoid blocking)
	go analyzeTaskAsync(id, taskPath)

	return c.JSON(fiber.Map{
		"message": "Task analysis started",
		"id": id,
	})
}

func getAgents(c *fiber.Ctx) error {
	rows, err := db.Query(`
		SELECT id, agent_name, current_task_id, current_task_title, 
		       status, started_at, last_heartbeat, resource_usage
		FROM swarm_manager.agent_status
		WHERE last_heartbeat > NOW() - INTERVAL '5 minutes'
		ORDER BY agent_name`)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to query agents",
		})
	}
	defer rows.Close()

	agents := []Agent{}
	for rows.Next() {
		var agent Agent
		var resourceJSON []byte
		err := rows.Scan(&agent.ID, &agent.Name, &agent.CurrentTaskID,
			&agent.CurrentTaskTitle, &agent.Status, &agent.StartedAt,
			&agent.LastHeartbeat, &resourceJSON)
		if err != nil {
			continue
		}

		if resourceJSON != nil {
			json.Unmarshal(resourceJSON, &agent.ResourceUsage)
		}
		agents = append(agents, agent)
	}

	return c.JSON(fiber.Map{
		"agents": agents,
		"count": len(agents),
	})
}

func agentHeartbeat(c *fiber.Ctx) error {
	name := c.Params("name")
	
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update or insert agent status
	resourceJSON, _ := json.Marshal(body["resource_usage"])
	
	_, err := db.Exec(`
		INSERT INTO swarm_manager.agent_status 
		(id, agent_name, current_task_id, current_task_title, status, last_heartbeat, resource_usage)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (agent_name) DO UPDATE
		SET current_task_id = $2, current_task_title = $3, status = $4,
		    last_heartbeat = NOW(), resource_usage = $5`,
		name, body["task_id"], body["task_title"], body["status"], resourceJSON)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to update agent status",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Heartbeat recorded",
		"agent": name,
	})
}

func getMetrics(c *fiber.Ctx) error {
	// Get various metrics
	metrics := make(map[string]interface{})

	// Task counts by status
	var taskCounts []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	
	rows, err := db.Query(`
		SELECT status, COUNT(*) 
		FROM swarm_manager.task_executions
		WHERE created_at > NOW() - INTERVAL '7 days'
		GROUP BY status`)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tc struct {
				Status string `json:"status"`
				Count  int    `json:"count"`
			}
			rows.Scan(&tc.Status, &tc.Count)
			taskCounts = append(taskCounts, tc)
		}
	}
	metrics["task_counts"] = taskCounts

	// Average execution time
	var avgDuration float64
	db.QueryRow(`
		SELECT AVG(duration_seconds)
		FROM swarm_manager.task_executions
		WHERE status = 'success' AND duration_seconds IS NOT NULL`).Scan(&avgDuration)
	metrics["avg_duration_seconds"] = avgDuration

	// Success rate
	var successRate float64
	db.QueryRow(`
		SELECT 
			CASE 
				WHEN COUNT(*) = 0 THEN 0
				ELSE CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) * 100
			END
		FROM swarm_manager.task_executions
		WHERE completed_at > NOW() - INTERVAL '7 days'`).Scan(&successRate)
	metrics["success_rate"] = successRate

	return c.JSON(metrics)
}

func getSuccessRate(c *fiber.Ctx) error {
	var successRate float64
	err := db.QueryRow(`
		SELECT 
			CASE 
				WHEN COUNT(*) = 0 THEN 0
				ELSE CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) * 100
			END
		FROM swarm_manager.task_executions
		WHERE completed_at > NOW() - INTERVAL '30 days'`).Scan(&successRate)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to calculate success rate",
		})
	}

	return c.JSON(fiber.Map{
		"success_rate": successRate,
		"period": "30_days",
	})
}

func getConfig(c *fiber.Ctx) error {
	config := Config{
		PriorityWeights: make(map[string]float64),
	}

	// Get configuration from database
	rows, err := db.Query(`
		SELECT setting_key, setting_value, setting_type
		FROM swarm_manager.configuration`)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to query configuration",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var key, value, settingType string
		rows.Scan(&key, &value, &settingType)

		switch key {
		case "max_concurrent_tasks":
			fmt.Sscanf(value, "%d", &config.MaxConcurrentTasks)
		case "min_backlog_size":
			fmt.Sscanf(value, "%d", &config.MinBacklogSize)
		case "task_scan_interval":
			fmt.Sscanf(value, "%d", &config.TaskScanInterval)
		case "yolo_mode":
			config.YoloMode = value == "true"
		case "max_cpu_percent":
			fmt.Sscanf(value, "%d", &config.MaxCPUPercent)
		case "max_memory_percent":
			fmt.Sscanf(value, "%d", &config.MaxMemoryPercent)
		case "max_claude_calls_per_hour":
			fmt.Sscanf(value, "%d", &config.MaxClaudeCallsPerHour)
		}
	}

	// Get priority weights
	rows, err = db.Query(`
		SELECT weight_type, weight_value
		FROM swarm_manager.priority_weights`)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var weightType string
			var weightValue float64
			rows.Scan(&weightType, &weightValue)
			config.PriorityWeights[weightType] = weightValue
		}
	}

	return c.JSON(config)
}

func updateConfig(c *fiber.Ctx) error {
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update configuration in database
	for key, value := range updates {
		valueStr := fmt.Sprintf("%v", value)
		_, err := db.Exec(`
			UPDATE swarm_manager.configuration
			SET setting_value = $1, updated_at = NOW()
			WHERE setting_key = $2`,
			valueStr, key)
		
		if err != nil {
			log.Printf("Failed to update config %s: %v", key, err)
		}
	}

	// Special handling for priority weights
	if weights, ok := updates["priority_weights"].(map[string]interface{}); ok {
		for weightType, weightValue := range weights {
			_, err := db.Exec(`
				UPDATE swarm_manager.priority_weights
				SET weight_value = $1, updated_at = NOW()
				WHERE weight_type = $2`,
				weightValue, weightType)
			
			if err != nil {
				log.Printf("Failed to update weight %s: %v", weightType, err)
			}
		}
	}

	return c.JSON(fiber.Map{
		"message": "Configuration updated",
		"updates": updates,
	})
}

func calculatePriority(c *fiber.Ctx) error {
	var estimates map[string]interface{}
	if err := c.BodyParser(&estimates); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get priority weights from database
	weights := make(map[string]float64)
	rows, err := db.Query(`
		SELECT weight_type, weight_value
		FROM swarm_manager.priority_weights`)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to query weights",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var weightType string
		var weightValue float64
		rows.Scan(&weightType, &weightValue)
		weights[weightType] = weightValue
	}

	// Calculate priority score
	urgencyMap := map[string]float64{
		"critical": 4.0,
		"high":     3.0,
		"medium":   2.0,
		"low":      1.0,
	}
	costMap := map[string]float64{
		"minimal":  1.0,
		"moderate": 2.0,
		"heavy":    3.0,
	}

	impact := getFloat(estimates["impact"], 5.0)
	urgency := urgencyMap[getString(estimates["urgency"], "medium")]
	successProb := getFloat(estimates["success_probability"], 0.5)
	resourceCost := costMap[getString(estimates["resource_cost"], "moderate")]
	cooldownHours := getFloat(estimates["cooldown_hours"], 0)

	priority := (impact * weights["impact"] * urgency * weights["urgency"] * successProb * weights["success"]) /
		(resourceCost * weights["cost"] * (1 + cooldownHours/24))

	return c.JSON(fiber.Map{
		"priority_score": priority,
		"weights_used": weights,
		"estimates_used": estimates,
	})
}

// Helper functions
func findTaskFile(id string) string {
	folders := []string{
		filepath.Join(tasksDir, "active"),
		filepath.Join(tasksDir, "staged"),
		filepath.Join(tasksDir, "backlog", "manual"),
		filepath.Join(tasksDir, "backlog", "generated"),
		filepath.Join(tasksDir, "completed"),
		filepath.Join(tasksDir, "failed"),
	}

	for _, folder := range folders {
		taskPath := filepath.Join(folder, fmt.Sprintf("%s.yaml", id))
		if _, err := os.Stat(taskPath); err == nil {
			return taskPath
		}
		taskPath = filepath.Join(folder, fmt.Sprintf("%s.yml", id))
		if _, err := os.Stat(taskPath); err == nil {
			return taskPath
		}
	}

	return ""
}

func getFloat(val interface{}, defaultVal float64) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	default:
		return defaultVal
	}
}

func getString(val interface{}, defaultVal string) string {
	if str, ok := val.(string); ok {
		return str
	}
	return defaultVal
}

// Task Execution Engine - Adapted from auto/ folder patterns

type TaskEvent struct {
	Type      string    `json:"type"`
	Task      string    `json:"task"`
	ID        string    `json:"id"`
	Timestamp time.Time `json:"ts"`
	PID       int       `json:"pid"`
	ExitCode  *int      `json:"exit_code,omitempty"`
	Duration  *int64    `json:"duration_sec,omitempty"`
	Scenario  string    `json:"scenario,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type ScenarioRegistry struct {
	Scenarios map[string]struct {
		CLI          string   `yaml:"cli"`
		Capabilities []string `yaml:"capabilities"`
		BestFor      []string `yaml:"best_for"`
	} `yaml:"scenarios"`
	DefaultScenario string `yaml:"default_scenario"`
	SelectionRules  []struct {
		IfContains []string `yaml:"if_contains"`
		Use        string   `yaml:"use"`
	} `yaml:"selection_rules"`
}

func executeTaskAsync(taskID, oldPath, activePath string) {
	start := time.Now()
	pid := os.Getpid()
	
	// Record start event
	recordTaskEvent(TaskEvent{
		Type:      "start",
		Task:      "execution",
		ID:        taskID,
		Timestamp: start,
		PID:       pid,
	})

	// Load task from file
	taskBytes, err := os.ReadFile(activePath)
	if err != nil {
		recordTaskEvent(TaskEvent{
			Type:      "error",
			Task:      "execution",
			ID:        taskID,
			Timestamp: time.Now(),
			PID:       pid,
			Error:     fmt.Sprintf("Failed to read task file: %v", err),
		})
		moveTaskToFailed(taskID, activePath, fmt.Sprintf("Failed to read task: %v", err))
		return
	}

	var task Task
	if err := yaml.Unmarshal(taskBytes, &task); err != nil {
		recordTaskEvent(TaskEvent{
			Type:      "error",
			Task:      "execution",
			ID:        taskID,
			Timestamp: time.Now(),
			PID:       pid,
			Error:     fmt.Sprintf("Failed to parse task YAML: %v", err),
		})
		moveTaskToFailed(taskID, activePath, fmt.Sprintf("Invalid YAML: %v", err))
		return
	}

	// Select appropriate scenario
	scenario := selectScenario(&task)
	
	// Build execution prompt
	prompt := buildTaskPrompt(&task, scenario)
	
	// Execute via claude-code (following auto/ pattern)
	var exitCode int
	var execErr error
	
	if scenario == "claude-code" {
		// Direct claude-code execution
		exitCode, execErr = executeViaClaude(prompt)
	} else {
		// Execute via scenario CLI
		exitCode, execErr = executeViaScenario(scenario, &task, prompt)
	}
	
	duration := int64(time.Since(start).Seconds())
	
	// Record completion event
	event := TaskEvent{
		Type:      "finish",
		Task:      "execution",
		ID:        taskID,
		Timestamp: time.Now(),
		PID:       pid,
		ExitCode:  &exitCode,
		Duration:  &duration,
		Scenario:  scenario,
	}
	
	if execErr != nil {
		event.Error = execErr.Error()
	}
	
	recordTaskEvent(event)
	
	// Move task based on result
	if exitCode == 0 && execErr == nil {
		moveTaskToCompleted(taskID, activePath)
	} else {
		errorMsg := fmt.Sprintf("Execution failed (exit %d)", exitCode)
		if execErr != nil {
			errorMsg = execErr.Error()
		}
		moveTaskToFailed(taskID, activePath, errorMsg)
	}
}

func selectScenario(task *Task) string {
	// Load scenario registry
	registryPath := filepath.Join(getBasePath(), "config", "scenario-registry.yaml")
	registryBytes, err := os.ReadFile(registryPath)
	if err != nil {
		log.Printf("Failed to load scenario registry, using claude-code: %v", err)
		return "claude-code"
	}
	
	var registry ScenarioRegistry
	if err := yaml.Unmarshal(registryBytes, &registry); err != nil {
		log.Printf("Failed to parse scenario registry, using claude-code: %v", err)
		return "claude-code"
	}
	
	// Check selection rules
	title := strings.ToLower(task.Title)
	description := strings.ToLower(task.Description)
	taskType := strings.ToLower(task.Type)
	searchText := title + " " + description + " " + taskType
	
	for _, rule := range registry.SelectionRules {
		for _, keyword := range rule.IfContains {
			if strings.Contains(searchText, strings.ToLower(keyword)) {
				return rule.Use
			}
		}
	}
	
	return registry.DefaultScenario
}

func buildTaskPrompt(task *Task, scenario string) string {
	// Load appropriate prompt template
	var promptFile string
	if scenario == "claude-code" {
		promptFile = "task-executor.md"
	} else {
		promptFile = "task-analyzer.md" // Use analyzer prompt for scenario selection
	}
	
	promptPath := filepath.Join(getBasePath(), "prompts", promptFile)
	promptTemplate, err := os.ReadFile(promptPath)
	if err != nil {
		log.Printf("Failed to load prompt template %s: %v", promptFile, err)
		// Fallback to basic prompt
		return fmt.Sprintf("Please complete this task: %s\n\nDescription: %s\nType: %s\nTarget: %s",
			task.Title, task.Description, task.Type, task.Target)
	}
	
	// Replace template variables (following auto/ pattern)
	prompt := string(promptTemplate)
	prompt = strings.ReplaceAll(prompt, "{{TASK_ID}}", task.ID)
	prompt = strings.ReplaceAll(prompt, "{{TASK_TITLE}}", task.Title)
	prompt = strings.ReplaceAll(prompt, "{{TASK_DESCRIPTION}}", task.Description)
	prompt = strings.ReplaceAll(prompt, "{{TASK_TYPE}}", task.Type)
	prompt = strings.ReplaceAll(prompt, "{{TASK_TARGET}}", task.Target)
	
	return prompt
}

func executeViaClaude(prompt string) (int, error) {
	// Execute using resource-claude-code (following auto/ pattern)
	cmd := exec.Command("resource-claude-code", "run", prompt)
	cmd.Dir = "/home/matthalloran8/Vrooli" // Set working directory
	
	output, err := cmd.CombinedOutput()
	exitCode := 0
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
		log.Printf("Claude execution failed: %v\nOutput: %s", err, string(output))
	}
	
	log.Printf("Claude execution completed with exit code %d", exitCode)
	return exitCode, err
}

func executeViaScenario(scenario string, task *Task, prompt string) (int, error) {
	// Execute using scenario CLI
	cmd := exec.Command(scenario, "run", "--task-id", task.ID, "--prompt", prompt)
	cmd.Dir = "/home/matthalloran8/Vrooli"
	
	output, err := cmd.CombinedOutput()
	exitCode := 0
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
		log.Printf("Scenario %s execution failed: %v\nOutput: %s", scenario, err, string(output))
	}
	
	log.Printf("Scenario %s execution completed with exit code %d", scenario, exitCode)
	return exitCode, err
}

func recordTaskEvent(event TaskEvent) {
	eventsPath := filepath.Join(getBasePath(), "logs", "events.ndjson")
	
	// Create logs directory if it doesn't exist
	logsDir := filepath.Dir(eventsPath)
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
		return
	}
	
	// Append event as JSON line
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}
	
	f, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open events file: %v", err)
		return
	}
	defer f.Close()
	
	if _, err := f.Write(append(eventJSON, '\n')); err != nil {
		log.Printf("Failed to write event: %v", err)
	}
}

func moveTaskToCompleted(taskID, activePath string) {
	completedDir := filepath.Join(getBasePath(), "tasks", "completed")
	if err := os.MkdirAll(completedDir, 0755); err != nil {
		log.Printf("Failed to create completed directory: %v", err)
		return
	}
	
	completedPath := filepath.Join(completedDir, filepath.Base(activePath))
	if err := os.Rename(activePath, completedPath); err != nil {
		log.Printf("Failed to move task to completed: %v", err)
	}
	
	// Update task with completion timestamp
	updateTaskTimestamp(completedPath, "completed_at")
}

func moveTaskToFailed(taskID, activePath, errorMsg string) {
	failedDir := filepath.Join(getBasePath(), "tasks", "failed")
	if err := os.MkdirAll(failedDir, 0755); err != nil {
		log.Printf("Failed to create failed directory: %v", err)
		return
	}
	
	failedPath := filepath.Join(failedDir, filepath.Base(activePath))
	if err := os.Rename(activePath, failedPath); err != nil {
		log.Printf("Failed to move task to failed: %v", err)
		return
	}
	
	// Add error info to task
	updateTaskWithError(failedPath, errorMsg)
}

func updateTaskTimestamp(taskPath, field string) {
	taskBytes, err := os.ReadFile(taskPath)
	if err != nil {
		return
	}
	
	var task map[string]interface{}
	if err := yaml.Unmarshal(taskBytes, &task); err != nil {
		return
	}
	
	task[field] = time.Now().Format(time.RFC3339)
	
	updatedBytes, err := yaml.Marshal(task)
	if err != nil {
		return
	}
	
	os.WriteFile(taskPath, updatedBytes, 0644)
}

func updateTaskWithError(taskPath, errorMsg string) {
	taskBytes, err := os.ReadFile(taskPath)
	if err != nil {
		return
	}
	
	var task map[string]interface{}
	if err := yaml.Unmarshal(taskBytes, &task); err != nil {
		return
	}
	
	task["failed_at"] = time.Now().Format(time.RFC3339)
	task["error"] = errorMsg
	
	// Add to attempts array
	attempts, ok := task["attempts"].([]interface{})
	if !ok {
		attempts = []interface{}{}
	}
	
	attempt := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"error":     errorMsg,
	}
	attempts = append(attempts, attempt)
	task["attempts"] = attempts
	
	updatedBytes, err := yaml.Marshal(task)
	if err != nil {
		return
	}
	
	os.WriteFile(taskPath, updatedBytes, 0644)
}

func analyzeTaskAsync(taskID, taskPath string) {
	start := time.Now()
	pid := os.Getpid()
	
	// Record start event
	recordTaskEvent(TaskEvent{
		Type:      "start",
		Task:      "analysis",
		ID:        taskID,
		Timestamp: start,
		PID:       pid,
	})

	// Load task from file
	taskBytes, err := os.ReadFile(taskPath)
	if err != nil {
		recordTaskEvent(TaskEvent{
			Type:      "error",
			Task:      "analysis",
			ID:        taskID,
			Timestamp: time.Now(),
			PID:       pid,
			Error:     fmt.Sprintf("Failed to read task file: %v", err),
		})
		return
	}

	var task Task
	if err := yaml.Unmarshal(taskBytes, &task); err != nil {
		recordTaskEvent(TaskEvent{
			Type:      "error",
			Task:      "analysis",
			ID:        taskID,
			Timestamp: time.Now(),
			PID:       pid,
			Error:     fmt.Sprintf("Failed to parse task YAML: %v", err),
		})
		return
	}

	// Build analysis prompt using the task-analyzer template
	prompt := buildAnalysisPrompt(&task)
	
	// Execute via claude-code
	exitCode, execErr := executeViaClaude(prompt)
	
	duration := int64(time.Since(start).Seconds())
	
	// Record completion event
	event := TaskEvent{
		Type:      "finish",
		Task:      "analysis",
		ID:        taskID,
		Timestamp: time.Now(),
		PID:       pid,
		ExitCode:  &exitCode,
		Duration:  &duration,
		Scenario:  "claude-code",
	}
	
	if execErr != nil {
		event.Error = execErr.Error()
	}
	
	recordTaskEvent(event)
	
	if exitCode == 0 && execErr == nil {
		// Analysis succeeded - the task file should now have updated priority estimates
		// Update the analyzed_at timestamp
		updateTaskTimestamp(taskPath, "analyzed_at")
		
		// Calculate priority score based on the new estimates
		updateTaskPriorityScore(taskPath)
	}
}

func buildAnalysisPrompt(task *Task) string {
	// Load the task analyzer prompt template
	promptPath := filepath.Join(getBasePath(), "prompts", "task-analyzer.md")
	promptTemplate, err := os.ReadFile(promptPath)
	if err != nil {
		log.Printf("Failed to load task analyzer prompt: %v", err)
		// Fallback to basic analysis prompt
		return fmt.Sprintf(`Please analyze this task and provide priority estimates:

Task ID: %s
Title: %s
Description: %s
Type: %s
Target: %s

Please provide estimates for:
- Impact (1-10): How much this task will improve the system
- Urgency (critical/high/medium/low): How time-sensitive this is
- Success Probability (0.0-1.0): Likelihood of successful completion
- Resource Cost (minimal/moderate/heavy): Expected resource usage

Update the task file with your estimates in YAML format.`, 
			task.ID, task.Title, task.Description, task.Type, task.Target)
	}
	
	// Replace template variables
	prompt := string(promptTemplate)
	prompt = strings.ReplaceAll(prompt, "{{TASK_ID}}", task.ID)
	prompt = strings.ReplaceAll(prompt, "{{TASK_TITLE}}", task.Title)
	prompt = strings.ReplaceAll(prompt, "{{TASK_DESCRIPTION}}", task.Description)
	prompt = strings.ReplaceAll(prompt, "{{TASK_TYPE}}", task.Type)
	prompt = strings.ReplaceAll(prompt, "{{TASK_TARGET}}", task.Target)
	
	// Add instruction to update the specific file
	taskPath := filepath.Join(getBasePath(), "tasks", "backlog", "manual", task.ID+".yaml")
	prompt += fmt.Sprintf("\n\nIMPORTANT: Please update the task file at %s with your priority estimates.", taskPath)
	
	return prompt
}

func updateTaskPriorityScore(taskPath string) {
	taskBytes, err := os.ReadFile(taskPath)
	if err != nil {
		return
	}
	
	var task map[string]interface{}
	if err := yaml.Unmarshal(taskBytes, &task); err != nil {
		return
	}
	
	// Get priority estimates
	estimates, ok := task["priority_estimates"].(map[string]interface{})
	if !ok {
		return
	}
	
	// Load priority weights from config
	weightsPath := filepath.Join(getBasePath(), "config", "priority-weights.yaml")
	weightsBytes, err := os.ReadFile(weightsPath)
	if err != nil {
		log.Printf("Failed to load priority weights: %v", err)
		return
	}
	
	var config map[string]interface{}
	if err := yaml.Unmarshal(weightsBytes, &config); err != nil {
		log.Printf("Failed to parse priority weights: %v", err)
		return
	}
	
	weights, ok := config["weights"].(map[string]interface{})
	if !ok {
		log.Printf("No weights found in priority config")
		return
	}
	
	// Calculate priority score: (Impact × Urgency × Success_Probability) / Resource_Cost
	impact := getFloat(estimates["impact"], 5.0)
	urgency := convertUrgencyToFloat(estimates["urgency"])
	successProb := getFloat(estimates["success_prob"], 0.5)
	resourceCost := convertResourceCostToFloat(estimates["resource_cost"])
	
	// Apply weights
	impactWeight := getFloat(weights["impact"], 1.0)
	urgencyWeight := getFloat(weights["urgency"], 0.8)
	successWeight := getFloat(weights["success"], 0.6)
	costWeight := getFloat(weights["cost"], 0.5)
	
	// Calculate weighted priority score
	numerator := (impact * impactWeight) * (urgency * urgencyWeight) * (successProb * successWeight)
	denominator := resourceCost * costWeight
	
	if denominator == 0 {
		denominator = 1
	}
	
	priorityScore := numerator / denominator
	task["priority_score"] = priorityScore
	
	// Save updated task
	updatedBytes, err := yaml.Marshal(task)
	if err != nil {
		log.Printf("Failed to marshal updated task: %v", err)
		return
	}
	
	if err := os.WriteFile(taskPath, updatedBytes, 0644); err != nil {
		log.Printf("Failed to write updated task: %v", err)
	}
}

func convertUrgencyToFloat(urgency interface{}) float64 {
	if str, ok := urgency.(string); ok {
		switch strings.ToLower(str) {
		case "critical":
			return 4.0
		case "high":
			return 3.0
		case "medium":
			return 2.0
		case "low":
			return 1.0
		}
	}
	return getFloat(urgency, 2.0) // default to medium
}

func convertResourceCostToFloat(cost interface{}) float64 {
	if str, ok := cost.(string); ok {
		switch strings.ToLower(str) {
		case "minimal":
			return 1.0
		case "moderate":
			return 2.0
		case "heavy":
			return 3.0
		}
	}
	return getFloat(cost, 2.0) // default to moderate
}

func getBasePath() string {
	// Get the swarm-manager directory path
	return "/home/matthalloran8/Vrooli/scenarios/swarm-manager"
}

// Agent Management System - Based on auto/ worker patterns

var (
	activeAgents = make(map[string]*Agent)
	agentsMutex  = sync.RWMutex{}
	maxAgents    = 3 // Following auto/ pattern
)

func startAgentSystem() {
	log.Println("Starting agent system...")
	
	// Start the configured number of agents
	for i := 0; i < maxAgents; i++ {
		agentID := fmt.Sprintf("agent-%d", i+1)
		startTime := time.Now()
		agent := &Agent{
			ID:            agentID,
			Name:          fmt.Sprintf("Worker Agent %d", i+1),
			Status:        "idle",
			StartedAt:     &startTime,
			LastHeartbeat: time.Now(),
			ResourceUsage: map[string]interface{}{"level": "minimal"},
		}
		
		// Register agent
		agentsMutex.Lock()
		activeAgents[agentID] = agent
		agentsMutex.Unlock()
		
		// Start agent goroutine
		go runAgent(agent)
	}
	
	// Start agent monitor
	go monitorAgents()
	
	log.Printf("Started %d agents", maxAgents)
}

func runAgent(agent *Agent) {
	log.Printf("Agent %s started", agent.Name)
	
	for {
		// Update heartbeat
		agentsMutex.Lock()
		agent.LastHeartbeat = time.Now()
		agentsMutex.Unlock()
		
		// Look for work in staged tasks (following priority order)
		taskPath := findHighestPriorityTask()
		if taskPath != "" {
			// Claim the task
			if claimTask(agent, taskPath) {
				// Execute the task
				executeTaskByAgent(agent, taskPath)
			}
		}
		
		// Wait before looking for more work (following auto/ pattern)
		time.Sleep(10 * time.Second)
	}
}

func findHighestPriorityTask() string {
	stagedDir := filepath.Join(getBasePath(), "tasks", "staged")
	
	files, err := filepath.Glob(filepath.Join(stagedDir, "*.yaml"))
	if err != nil {
		return ""
	}
	
	if len(files) == 0 {
		return ""
	}
	
	// Find task with highest priority score
	var bestPath string
	var highestPriority float64 = -1
	
	for _, file := range files {
		taskBytes, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		var task map[string]interface{}
		if err := yaml.Unmarshal(taskBytes, &task); err != nil {
			continue
		}
		
		priority := getFloat(task["priority_score"], 0)
		if priority > highestPriority {
			highestPriority = priority
			bestPath = file
		}
	}
	
	return bestPath
}

func claimTask(agent *Agent, taskPath string) bool {
	// Move task from staged to active (atomic operation)
	activeDir := filepath.Join(getBasePath(), "tasks", "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		log.Printf("Failed to create active directory: %v", err)
		return false
	}
	
	fileName := filepath.Base(taskPath)
	activePath := filepath.Join(activeDir, fileName)
	
	// Attempt atomic move (acts as lock)
	if err := os.Rename(taskPath, activePath); err != nil {
		// Task was likely claimed by another agent
		return false
	}
	
	// Update agent status
	agentsMutex.Lock()
	agent.Status = "working"
	
	// Load task to get title
	if taskBytes, err := os.ReadFile(activePath); err == nil {
		var task Task
		if err := yaml.Unmarshal(taskBytes, &task); err == nil {
			agent.CurrentTaskID = &task.ID
			agent.CurrentTaskTitle = &task.Title
		}
	}
	agentsMutex.Unlock()
	
	log.Printf("Agent %s claimed task %s", agent.Name, fileName)
	return true
}

func executeTaskByAgent(agent *Agent, activePath string) {
	fileName := filepath.Base(activePath)
	taskID := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	
	log.Printf("Agent %s executing task %s", agent.Name, taskID)
	
	// Update agent resource usage during execution
	agentsMutex.Lock()
	agent.ResourceUsage = map[string]interface{}{"level": "moderate"}
	agentsMutex.Unlock()
	
	// Use the existing execution engine
	executeTaskAsync(taskID, "", activePath)
	
	// Reset agent status
	agentsMutex.Lock()
	agent.Status = "idle"
	agent.CurrentTaskID = nil
	agent.CurrentTaskTitle = nil
	agent.ResourceUsage = map[string]interface{}{"level": "minimal"}
	agentsMutex.Unlock()
	
	log.Printf("Agent %s completed task %s", agent.Name, taskID)
}

func monitorAgents() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		agentsMutex.RLock()
		for _, agent := range activeAgents {
			// Update database with agent status
			_, err := db.Exec(`
				INSERT INTO swarm_manager.agent_status 
				(id, agent_name, current_task_id, current_task_title, status, started_at, last_heartbeat, resource_usage)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (id) DO UPDATE SET
					current_task_id = EXCLUDED.current_task_id,
					current_task_title = EXCLUDED.current_task_title,
					status = EXCLUDED.status,
					last_heartbeat = EXCLUDED.last_heartbeat,
					resource_usage = EXCLUDED.resource_usage`,
				agent.ID, agent.Name, agent.CurrentTaskID, agent.CurrentTaskTitle, 
				agent.Status, agent.StartedAt, agent.LastHeartbeat, agent.ResourceUsage)
			
			if err != nil {
				log.Printf("Failed to update agent status: %v", err)
			}
		}
		agentsMutex.RUnlock()
	}
}

func stopAgentSystem() {
	log.Println("Stopping agent system...")
	// In a full implementation, we'd have a context to cancel agent goroutines
	// For now, agents will stop when the process exits
}

// Problem management handlers

func scanProblems(c *fiber.Ctx) error {
	start := time.Now()
	
	var req ProblemScanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	if req.ScanPath == "" {
		req.ScanPath = "/home/matthalloran8/Vrooli"
	}
	
	log.Printf("Scanning for problems in: %s", req.ScanPath)
	
	// Find all PROBLEMS.md files
	problemFiles, err := findProblemFiles(req.ScanPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to scan for problem files", "details": err.Error()})
	}
	
	var allProblems []Problem
	var tasksCreated int
	
	for _, file := range problemFiles {
		problems, err := parseProblemsFromFile(file)
		if err != nil {
			log.Printf("Error parsing %s: %v", file, err)
			continue
		}
		
		for _, problem := range problems {
			// Store problem in database
			err := storeProblem(problem)
			if err != nil {
				log.Printf("Error storing problem %s: %v", problem.ID, err)
				continue
			}
			
			// Create task for critical/high problems if yolo mode is on
			config, _ := getCurrentConfig()
			if config.YoloMode && (problem.Severity == "critical" || problem.Severity == "high") {
				taskCreated := createTaskFromProblem(problem)
				if taskCreated {
					tasksCreated++
				}
			}
		}
		
		allProblems = append(allProblems, problems...)
	}
	
	scanTime := time.Since(start).Seconds()
	
	response := ProblemScanResponse{
		ProblemsFound: len(allProblems),
		TasksCreated:  tasksCreated,
		ScanTime:      scanTime,
		NewProblems:   extractProblemIDs(allProblems),
	}
	
	return c.JSON(response)
}

func getProblems(c *fiber.Ctx) error {
	filter := c.Query("filter", "all")
	
	var problems []Problem
	var query string
	var args []interface{}
	
	baseQuery := `SELECT id, title, description, severity, frequency, impact, status, 
	                     discovered_at, discovered_by, last_occurrence, source_file,
	                     affected_components, symptoms, evidence, related_issues,
	                     priority_estimates, resolution, resolved_at, resolved_by, tasks_created
	              FROM problems`
	
	switch filter {
	case "active":
		query = baseQuery + " WHERE status = 'active'"
	case "critical":
		query = baseQuery + " WHERE severity = 'critical'"
	case "resolved":
		query = baseQuery + " WHERE status = 'resolved'"
	default:
		query = baseQuery
	}
	
	query += " ORDER BY discovered_at DESC LIMIT 100"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch problems", "details": err.Error()})
	}
	defer rows.Close()
	
	for rows.Next() {
		var problem Problem
		var affectedComponentsJSON, symptomsJSON, evidenceJSON, relatedIssuesJSON, priorityEstimatesJSON, tasksCreatedJSON []byte
		
		err := rows.Scan(
			&problem.ID, &problem.Title, &problem.Description, &problem.Severity, 
			&problem.Frequency, &problem.Impact, &problem.Status,
			&problem.DiscoveredAt, &problem.DiscoveredBy, &problem.LastOccurrence, &problem.SourceFile,
			&affectedComponentsJSON, &symptomsJSON, &evidenceJSON, &relatedIssuesJSON,
			&priorityEstimatesJSON, &problem.Resolution, &problem.ResolvedAt, &problem.ResolvedBy, &tasksCreatedJSON,
		)
		if err != nil {
			log.Printf("Error scanning problem row: %v", err)
			continue
		}
		
		// Parse JSON fields
		json.Unmarshal(affectedComponentsJSON, &problem.AffectedComponents)
		json.Unmarshal(symptomsJSON, &problem.Symptoms)
		json.Unmarshal(evidenceJSON, &problem.Evidence)
		json.Unmarshal(relatedIssuesJSON, &problem.RelatedIssues)
		json.Unmarshal(priorityEstimatesJSON, &problem.PriorityEstimates)
		json.Unmarshal(tasksCreatedJSON, &problem.TasksCreated)
		
		problems = append(problems, problem)
	}
	
	return c.JSON(fiber.Map{"problems": problems})
}

func getProblem(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var problem Problem
	var affectedComponentsJSON, symptomsJSON, evidenceJSON, relatedIssuesJSON, priorityEstimatesJSON, tasksCreatedJSON []byte
	
	query := `SELECT id, title, description, severity, frequency, impact, status, 
	                 discovered_at, discovered_by, last_occurrence, source_file,
	                 affected_components, symptoms, evidence, related_issues,
	                 priority_estimates, resolution, resolved_at, resolved_by, tasks_created
	          FROM problems WHERE id = $1`
	
	err := db.QueryRow(query, id).Scan(
		&problem.ID, &problem.Title, &problem.Description, &problem.Severity, 
		&problem.Frequency, &problem.Impact, &problem.Status,
		&problem.DiscoveredAt, &problem.DiscoveredBy, &problem.LastOccurrence, &problem.SourceFile,
		&affectedComponentsJSON, &symptomsJSON, &evidenceJSON, &relatedIssuesJSON,
		&priorityEstimatesJSON, &problem.Resolution, &problem.ResolvedAt, &problem.ResolvedBy, &tasksCreatedJSON,
	)
	
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Problem not found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error", "details": err.Error()})
	}
	
	// Parse JSON fields
	json.Unmarshal(affectedComponentsJSON, &problem.AffectedComponents)
	json.Unmarshal(symptomsJSON, &problem.Symptoms)
	json.Unmarshal(evidenceJSON, &problem.Evidence)
	json.Unmarshal(relatedIssuesJSON, &problem.RelatedIssues)
	json.Unmarshal(priorityEstimatesJSON, &problem.PriorityEstimates)
	json.Unmarshal(tasksCreatedJSON, &problem.TasksCreated)
	
	return c.JSON(problem)
}

func resolveProblem(c *fiber.Ctx) error {
	id := c.Params("id")
	
	type ResolveRequest struct {
		Resolution string `json:"resolution"`
		ResolvedBy string `json:"resolved_by"`
	}
	
	var req ResolveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	
	if req.ResolvedBy == "" {
		req.ResolvedBy = "manual"
	}
	
	now := time.Now()
	query := `UPDATE problems 
	          SET status = 'resolved', resolution = $1, resolved_at = $2, resolved_by = $3
	          WHERE id = $4`
	
	result, err := db.Exec(query, req.Resolution, now, req.ResolvedBy, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error", "details": err.Error()})
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Problem not found"})
	}
	
	return c.JSON(fiber.Map{"message": "Problem resolved successfully"})
}

// Helper functions for problem management

func findProblemFiles(scanPath string) ([]string, error) {
	var problemFiles []string
	
	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}
		
		if info.Name() == "PROBLEMS.md" {
			problemFiles = append(problemFiles, path)
		}
		
		// Limit to prevent excessive scanning
		if len(problemFiles) >= 50 {
			return filepath.SkipDir
		}
		
		return nil
	})
	
	return problemFiles, err
}

func parseProblemsFromFile(filename string) ([]Problem, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var problems []Problem
	
	// Simple parser - look for ACTIVEPROBLEM embedding markers
	lines := strings.Split(string(content), "\n")
	var currentProblem *Problem
	var inActiveProblem bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "<!-- EMBED:ACTIVEPROBLEM:START -->") {
			inActiveProblem = true
			currentProblem = &Problem{
				ID:           fmt.Sprintf("prob-%d", time.Now().Unix()),
				DiscoveredAt: time.Now(),
				DiscoveredBy: "problem-scanner",
				SourceFile:   filename,
				Status:       "active",
			}
			continue
		}
		
		if strings.Contains(line, "<!-- EMBED:ACTIVEPROBLEM:END -->") {
			if currentProblem != nil && currentProblem.Title != "" {
				problems = append(problems, *currentProblem)
			}
			inActiveProblem = false
			currentProblem = nil
			continue
		}
		
		if inActiveProblem && currentProblem != nil {
			// Parse problem fields
			if strings.HasPrefix(line, "### ") {
				currentProblem.Title = strings.TrimPrefix(line, "### ")
			} else if strings.HasPrefix(line, "**Severity:**") {
				severity := strings.TrimSpace(strings.TrimPrefix(line, "**Severity:**"))
				severity = strings.Trim(severity, "[]")
				if len(severity) > 0 {
					currentProblem.Severity = strings.Split(severity, "|")[0]
				}
			} else if strings.HasPrefix(line, "**Frequency:**") {
				frequency := strings.TrimSpace(strings.TrimPrefix(line, "**Frequency:**"))
				frequency = strings.Trim(frequency, "[]")
				if len(frequency) > 0 {
					currentProblem.Frequency = strings.Split(frequency, "|")[0]
				}
			} else if strings.HasPrefix(line, "**Impact:**") {
				impact := strings.TrimSpace(strings.TrimPrefix(line, "**Impact:**"))
				impact = strings.Trim(impact, "[]")
				if len(impact) > 0 {
					currentProblem.Impact = strings.Split(impact, "|")[0]
				}
			}
		}
	}
	
	return problems, nil
}

func storeProblem(problem Problem) error {
	// Convert slices and maps to JSON for storage
	affectedComponents, _ := json.Marshal(problem.AffectedComponents)
	symptoms, _ := json.Marshal(problem.Symptoms)
	evidence, _ := json.Marshal(problem.Evidence)
	relatedIssues, _ := json.Marshal(problem.RelatedIssues)
	priorityEstimates, _ := json.Marshal(problem.PriorityEstimates)
	tasksCreated, _ := json.Marshal(problem.TasksCreated)
	
	query := `INSERT INTO problems (
		id, title, description, severity, frequency, impact, status,
		discovered_at, discovered_by, last_occurrence, source_file,
		affected_components, symptoms, evidence, related_issues,
		priority_estimates, tasks_created
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title,
		description = EXCLUDED.description,
		severity = EXCLUDED.severity,
		frequency = EXCLUDED.frequency,
		impact = EXCLUDED.impact,
		last_occurrence = EXCLUDED.last_occurrence,
		affected_components = EXCLUDED.affected_components,
		symptoms = EXCLUDED.symptoms,
		evidence = EXCLUDED.evidence,
		related_issues = EXCLUDED.related_issues,
		priority_estimates = EXCLUDED.priority_estimates`
	
	_, err := db.Exec(query,
		problem.ID, problem.Title, problem.Description, problem.Severity,
		problem.Frequency, problem.Impact, problem.Status,
		problem.DiscoveredAt, problem.DiscoveredBy, problem.LastOccurrence, problem.SourceFile,
		string(affectedComponents), string(symptoms), string(evidence), string(relatedIssues),
		string(priorityEstimates), string(tasksCreated))
		
	return err
}

func createTaskFromProblem(problem Problem) bool {
	// Create a task based on the problem
	taskID := fmt.Sprintf("task-prob-%s-%d", problem.Severity, time.Now().Unix())
	
	task := Task{
		ID:          taskID,
		Title:       fmt.Sprintf("Resolve %s problem: %s", problem.Severity, problem.Title),
		Description: problem.Description,
		Type:        "problem-resolution",
		Target:      problem.SourceFile,
		PriorityEstimates: map[string]interface{}{
			"impact":       severityToImpact(problem.Severity),
			"urgency":      frequencyToUrgency(problem.Frequency),
			"success_prob": 0.7, // Default assumption
			"resource_cost": impactToResourceCost(problem.Impact),
		},
		Dependencies: []string{},
		Blockers:     []string{},
		CreatedAt:    time.Now(),
		CreatedBy:    "problem-scanner",
		Attempts:     []interface{}{},
		Notes:        fmt.Sprintf("Auto-generated from problem: %s", problem.ID),
	}
	
	// Save task to appropriate directory
	var taskDir string
	if problem.Severity == "critical" || problem.Severity == "high" {
		taskDir = filepath.Join(tasksDir, "backlog", "generated")
	} else {
		taskDir = filepath.Join(tasksDir, "backlog", "manual")
	}
	
	err := saveTaskToFile(task, taskDir)
	if err != nil {
		log.Printf("Failed to create task from problem %s: %v", problem.ID, err)
		return false
	}
	
	// Update problem with created task
	problem.TasksCreated = append(problem.TasksCreated, taskID)
	storeProblem(problem)
	
	log.Printf("Created task %s from problem %s", taskID, problem.ID)
	return true
}

func severityToImpact(severity string) int {
	switch severity {
	case "critical":
		return 10
	case "high":
		return 8
	case "medium":
		return 5
	case "low":
		return 2
	default:
		return 5
	}
}

func frequencyToUrgency(frequency string) string {
	switch frequency {
	case "constant":
		return "critical"
	case "frequent":
		return "high"
	case "occasional":
		return "medium"
	case "rare":
		return "low"
	default:
		return "medium"
	}
}

func impactToResourceCost(impact string) string {
	switch impact {
	case "system_down":
		return "heavy"
	case "degraded_performance":
		return "moderate"
	case "user_impact":
		return "moderate"
	case "cosmetic":
		return "minimal"
	default:
		return "moderate"
	}
}

func extractProblemIDs(problems []Problem) []string {
	ids := make([]string, len(problems))
	for i, problem := range problems {
		ids[i] = problem.ID
	}
	return ids
}

func getCurrentConfig() (Config, error) {
	// Default configuration
	config := Config{
		MaxConcurrentTasks:    5,
		MinBacklogSize:        10,
		TaskScanInterval:      300,
		YoloMode:              false,
		MaxCPUPercent:         70,
		MaxMemoryPercent:      70,
		MaxClaudeCallsPerHour: 100,
		PriorityWeights: map[string]float64{
			"impact":  1.0,
			"urgency": 0.8,
			"success": 0.6,
			"cost":    0.5,
		},
	}
	
	// Query database for actual config values
	rows, err := db.Query(`SELECT setting_key, setting_value, setting_type FROM configuration`)
	if err != nil {
		return config, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var key, value, settingType string
		if err := rows.Scan(&key, &value, &settingType); err != nil {
			continue
		}
		
		// Update config based on database values
		switch key {
		case "yolo_mode":
			config.YoloMode = (value == "true")
		case "max_concurrent_tasks":
			if val, err := strconv.Atoi(value); err == nil {
				config.MaxConcurrentTasks = val
			}
		case "min_backlog_size":
			if val, err := strconv.Atoi(value); err == nil {
				config.MinBacklogSize = val
			}
		}
	}
	
	return config, nil
}

func saveTaskToFile(task Task, taskDir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}
	
	// Create task file path
	filename := fmt.Sprintf("%s.yaml", task.ID)
	filepath := filepath.Join(taskDir, filename)
	
	// Convert task to YAML
	yamlData, err := yaml.Marshal(task)
	if err != nil {
		return err
	}
	
	// Write to file
	return ioutil.WriteFile(filepath, yamlData, 0644)
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
		"code": code,
	})
}