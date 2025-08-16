package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
    
    _ "github.com/lib/pq"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/go-redis/redis/v8"
    "context"
)

type Agent struct {
    ID            string                 `json:"id"`
    Name          string                 `json:"name"`
    Type          string                 `json:"type"`
    Status        string                 `json:"status"`
    Description   string                 `json:"description,omitempty"`
    LastHeartbeat *time.Time            `json:"last_heartbeat,omitempty"`
    Capabilities  []string              `json:"capabilities"`
    Metrics       map[string]interface{} `json:"metrics"`
    Configuration map[string]interface{} `json:"configuration,omitempty"`
    HealthScore   int                   `json:"health_score"`
}

type OrchestrationRequest struct {
    Task        string                 `json:"task"`
    Context     map[string]interface{} `json:"context"`
    Constraints map[string]interface{} `json:"constraints"`
}

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

var (
    db       *sql.DB
    rdb      *redis.Client
    upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true // Allow all origins in development
        },
    }
    ctx = context.Background()
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8100"
    }
    
    // Initialize database connection
    initDB()
    defer db.Close()
    
    // Initialize Redis connection
    initRedis()
    defer rdb.Close()
    
    // Setup routes
    router := mux.NewRouter()
    
    // API endpoints
    router.HandleFunc("/health", healthHandler).Methods("GET")
    router.HandleFunc("/api/agents", agentsHandler).Methods("GET", "POST")
    router.HandleFunc("/api/agents/{id}", agentHandler).Methods("GET", "PUT", "DELETE")
    router.HandleFunc("/api/agents/{id}/heartbeat", heartbeatHandler).Methods("POST")
    router.HandleFunc("/api/agents/{id}/metrics", metricsHandler).Methods("GET", "POST")
    router.HandleFunc("/api/agents/{id}/logs", logsHandler).Methods("GET", "POST")
    router.HandleFunc("/api/agents/{id}/control", controlHandler).Methods("POST")
    router.HandleFunc("/api/orchestrate", orchestrateHandler).Methods("POST")
    router.HandleFunc("/api/health/system", systemHealthHandler).Methods("GET")
    router.HandleFunc("/ws", wsHandler)
    
    // Enable CORS
    router.Use(corsMiddleware)
    
    log.Printf("Agent Dashboard API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, router); err != nil {
        log.Fatal(err)
    }
}

func initDB() {
    dbURL := os.Getenv("POSTGRES_URL")
    if dbURL == "" {
        dbURL = "postgres://postgres:postgres@localhost:5432/agent_dashboard?sslmode=disable"
    }
    
    var err error
    db, err = sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    if err = db.Ping(); err != nil {
        log.Fatal("Failed to ping database:", err)
    }
    
    log.Println("Database connected successfully")
}

func initRedis() {
    redisURL := os.Getenv("REDIS_URL")
    if redisURL == "" {
        redisURL = "redis://localhost:6379"
    }
    
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        log.Fatal("Failed to parse Redis URL:", err)
    }
    
    rdb = redis.NewClient(opt)
    
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }
    
    log.Println("Redis connected successfully")
}

func corsMiddleware(next http.Handler) http.Handler {
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := APIResponse{
        Success: true,
        Data: map[string]interface{}{
            "status":  "healthy",
            "service": "agent-dashboard-api",
            "version": "2.0.0",
            "database": db.Ping() == nil,
            "redis":   rdb.Ping(ctx).Err() == nil,
        },
    }
    jsonResponse(w, response, http.StatusOK)
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        agents, err := getAgentsFromDB()
        if err != nil {
            errorResponse(w, "Failed to fetch agents: "+err.Error(), http.StatusInternalServerError)
            return
        }
        response := APIResponse{
            Success: true,
            Data:    agents,
        }
        jsonResponse(w, response, http.StatusOK)
        
    case "POST":
        var agent Agent
        if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
            errorResponse(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        
        id, err := createAgent(agent)
        if err != nil {
            errorResponse(w, "Failed to create agent: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        agent.ID = id
        response := APIResponse{
            Success: true,
            Data:    agent,
        }
        jsonResponse(w, response, http.StatusCreated)
    }
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    agentID := vars["id"]
    
    switch r.Method {
    case "GET":
        agent, err := getAgentByID(agentID)
        if err != nil {
            errorResponse(w, "Agent not found", http.StatusNotFound)
            return
        }
        response := APIResponse{
            Success: true,
            Data:    agent,
        }
        jsonResponse(w, response, http.StatusOK)
        
    case "PUT":
        var updates map[string]interface{}
        if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
            errorResponse(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        
        if err := updateAgent(agentID, updates); err != nil {
            errorResponse(w, "Failed to update agent: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        response := APIResponse{
            Success: true,
            Data:    map[string]string{"message": "Agent updated successfully"},
        }
        jsonResponse(w, response, http.StatusOK)
        
    case "DELETE":
        if err := deleteAgent(agentID); err != nil {
            errorResponse(w, "Failed to delete agent: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        response := APIResponse{
            Success: true,
            Data:    map[string]string{"message": "Agent deleted successfully"},
        }
        jsonResponse(w, response, http.StatusOK)
    }
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    agentID := vars["id"]
    
    var data map[string]interface{}
    json.NewDecoder(r.Body).Decode(&data)
    
    if err := updateHeartbeat(agentID, data); err != nil {
        errorResponse(w, "Failed to update heartbeat: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Publish heartbeat to Redis for real-time updates
    heartbeatData := map[string]interface{}{
        "agent_id":  agentID,
        "timestamp": time.Now(),
        "data":      data,
    }
    rdb.Publish(ctx, "agent_heartbeats", mustMarshal(heartbeatData))
    
    response := APIResponse{
        Success: true,
        Data:    map[string]string{"message": "Heartbeat recorded"},
    }
    jsonResponse(w, response, http.StatusOK)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    agentID := vars["id"]
    
    switch r.Method {
    case "GET":
        metrics, err := getAgentMetrics(agentID)
        if err != nil {
            errorResponse(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
            return
        }
        response := APIResponse{
            Success: true,
            Data:    metrics,
        }
        jsonResponse(w, response, http.StatusOK)
        
    case "POST":
        var metrics map[string]interface{}
        if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
            errorResponse(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        
        if err := recordMetrics(agentID, metrics); err != nil {
            errorResponse(w, "Failed to record metrics: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        response := APIResponse{
            Success: true,
            Data:    map[string]string{"message": "Metrics recorded"},
        }
        jsonResponse(w, response, http.StatusOK)
    }
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    agentID := vars["id"]
    
    switch r.Method {
    case "GET":
        logs, err := getAgentLogs(agentID)
        if err != nil {
            errorResponse(w, "Failed to fetch logs: "+err.Error(), http.StatusInternalServerError)
            return
        }
        response := APIResponse{
            Success: true,
            Data:    logs,
        }
        jsonResponse(w, response, http.StatusOK)
        
    case "POST":
        var logEntry map[string]interface{}
        if err := json.NewDecoder(r.Body).Decode(&logEntry); err != nil {
            errorResponse(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        
        if err := addLogEntry(agentID, logEntry); err != nil {
            errorResponse(w, "Failed to add log entry: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        response := APIResponse{
            Success: true,
            Data:    map[string]string{"message": "Log entry added"},
        }
        jsonResponse(w, response, http.StatusOK)
    }
}

func controlHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    agentID := vars["id"]
    
    var control struct {
        Action string                 `json:"action"`
        Params map[string]interface{} `json:"params"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&control); err != nil {
        errorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Execute control action
    result, err := executeControl(agentID, control.Action, control.Params)
    if err != nil {
        errorResponse(w, "Failed to execute control: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    response := APIResponse{
        Success: true,
        Data:    result,
    }
    jsonResponse(w, response, http.StatusOK)
}

func orchestrateHandler(w http.ResponseWriter, r *http.Request) {
    var request OrchestrationRequest
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        errorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Call n8n workflow for orchestration
    orchestrationID := fmt.Sprintf("orch-%d", time.Now().Unix())
    
    // Store orchestration request in Redis for processing
    orchData := map[string]interface{}{
        "id":          orchestrationID,
        "task":        request.Task,
        "context":     request.Context,
        "constraints": request.Constraints,
        "status":      "queued",
        "created_at":  time.Now(),
    }
    
    rdb.Set(ctx, "orchestration:"+orchestrationID, mustMarshal(orchData), 1*time.Hour)
    rdb.Publish(ctx, "orchestration_requests", mustMarshal(orchData))
    
    response := APIResponse{
        Success: true,
        Data: map[string]interface{}{
            "orchestration_id": orchestrationID,
            "status":          "queued",
            "message":         "Orchestration plan queued for processing",
        },
    }
    jsonResponse(w, response, http.StatusOK)
}

func systemHealthHandler(w http.ResponseWriter, r *http.Request) {
    // Get cached health status from Redis
    healthData, err := rdb.Get(ctx, "agent_health_status").Result()
    if err != nil {
        // Fallback to database query
        health, err := getSystemHealth()
        if err != nil {
            errorResponse(w, "Failed to fetch system health: "+err.Error(), http.StatusInternalServerError)
            return
        }
        response := APIResponse{
            Success: true,
            Data:    health,
        }
        jsonResponse(w, response, http.StatusOK)
        return
    }
    
    var health map[string]interface{}
    json.Unmarshal([]byte(healthData), &health)
    
    response := APIResponse{
        Success: true,
        Data:    health,
    }
    jsonResponse(w, response, http.StatusOK)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket upgrade failed:", err)
        return
    }
    defer conn.Close()
    
    // Subscribe to Redis channels for real-time updates
    pubsub := rdb.Subscribe(ctx, "agent_heartbeats", "agent_health_updates", "orchestration_updates")
    defer pubsub.Close()
    
    ch := pubsub.Channel()
    
    for msg := range ch {
        if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
            log.Println("WebSocket write error:", err)
            break
        }
    }
}

// Database helper functions
func getAgentsFromDB() ([]Agent, error) {
    query := `
        SELECT id, name, type, status, description, last_heartbeat, 
               capabilities::text, metrics::text, configuration::text
        FROM agent_dashboard.agents
        WHERE status != 'terminated'
        ORDER BY type, name
    `
    
    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var agents []Agent
    for rows.Next() {
        var agent Agent
        var capabilities, metrics, config sql.NullString
        
        err := rows.Scan(
            &agent.ID, &agent.Name, &agent.Type, &agent.Status,
            &agent.Description, &agent.LastHeartbeat,
            &capabilities, &metrics, &config,
        )
        if err != nil {
            continue
        }
        
        if capabilities.Valid {
            json.Unmarshal([]byte(capabilities.String), &agent.Capabilities)
        }
        if metrics.Valid {
            json.Unmarshal([]byte(metrics.String), &agent.Metrics)
        }
        if config.Valid {
            json.Unmarshal([]byte(config.String), &agent.Configuration)
        }
        
        agents = append(agents, agent)
    }
    
    return agents, nil
}

func getAgentByID(id string) (*Agent, error) {
    var agent Agent
    var capabilities, metrics, config sql.NullString
    
    query := `
        SELECT id, name, type, status, description, last_heartbeat,
               capabilities::text, metrics::text, configuration::text
        FROM agent_dashboard.agents
        WHERE id = $1
    `
    
    err := db.QueryRow(query, id).Scan(
        &agent.ID, &agent.Name, &agent.Type, &agent.Status,
        &agent.Description, &agent.LastHeartbeat,
        &capabilities, &metrics, &config,
    )
    
    if err != nil {
        return nil, err
    }
    
    if capabilities.Valid {
        json.Unmarshal([]byte(capabilities.String), &agent.Capabilities)
    }
    if metrics.Valid {
        json.Unmarshal([]byte(metrics.String), &agent.Metrics)
    }
    if config.Valid {
        json.Unmarshal([]byte(config.String), &agent.Configuration)
    }
    
    return &agent, nil
}

func createAgent(agent Agent) (string, error) {
    var id string
    query := `
        INSERT INTO agent_dashboard.agents (name, type, status, description, capabilities, metrics, configuration)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb)
        RETURNING id
    `
    
    err := db.QueryRow(
        query,
        agent.Name, agent.Type, agent.Status, agent.Description,
        mustMarshal(agent.Capabilities), mustMarshal(agent.Metrics), mustMarshal(agent.Configuration),
    ).Scan(&id)
    
    return id, err
}

func updateAgent(id string, updates map[string]interface{}) error {
    query := `
        UPDATE agent_dashboard.agents
        SET updated_at = CURRENT_TIMESTAMP
        WHERE id = $1
    `
    _, err := db.Exec(query, id)
    return err
}

func deleteAgent(id string) error {
    query := `UPDATE agent_dashboard.agents SET status = 'terminated' WHERE id = $1`
    _, err := db.Exec(query, id)
    return err
}

func updateHeartbeat(agentID string, data map[string]interface{}) error {
    query := `
        UPDATE agent_dashboard.agents
        SET last_heartbeat = CURRENT_TIMESTAMP,
            metrics = $2::jsonb
        WHERE id = $1
    `
    _, err := db.Exec(query, agentID, mustMarshal(data))
    return err
}

func getAgentMetrics(agentID string) ([]map[string]interface{}, error) {
    query := `
        SELECT metric_type, value, unit, tags::text, recorded_at
        FROM agent_dashboard.agent_metrics
        WHERE agent_id = $1
        ORDER BY recorded_at DESC
        LIMIT 100
    `
    
    rows, err := db.Query(query, agentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var metrics []map[string]interface{}
    for rows.Next() {
        var metricType string
        var value float64
        var unit sql.NullString
        var tags sql.NullString
        var recordedAt time.Time
        
        err := rows.Scan(&metricType, &value, &unit, &tags, &recordedAt)
        if err != nil {
            continue
        }
        
        metric := map[string]interface{}{
            "type":        metricType,
            "value":       value,
            "recorded_at": recordedAt,
        }
        
        if unit.Valid {
            metric["unit"] = unit.String
        }
        if tags.Valid {
            var tagData map[string]interface{}
            json.Unmarshal([]byte(tags.String), &tagData)
            metric["tags"] = tagData
        }
        
        metrics = append(metrics, metric)
    }
    
    return metrics, nil
}

func recordMetrics(agentID string, metrics map[string]interface{}) error {
    for metricType, value := range metrics {
        query := `
            INSERT INTO agent_dashboard.agent_metrics (agent_id, metric_type, value, unit)
            VALUES ($1, $2, $3, $4)
        `
        _, err := db.Exec(query, agentID, metricType, value, "units")
        if err != nil {
            return err
        }
    }
    return nil
}

func getAgentLogs(agentID string) ([]map[string]interface{}, error) {
    query := `
        SELECT level, message, context::text, timestamp
        FROM agent_dashboard.agent_logs
        WHERE agent_id = $1
        ORDER BY timestamp DESC
        LIMIT 100
    `
    
    rows, err := db.Query(query, agentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var logs []map[string]interface{}
    for rows.Next() {
        var level, message string
        var context sql.NullString
        var timestamp time.Time
        
        err := rows.Scan(&level, &message, &context, &timestamp)
        if err != nil {
            continue
        }
        
        log := map[string]interface{}{
            "level":     level,
            "message":   message,
            "timestamp": timestamp,
        }
        
        if context.Valid {
            var contextData map[string]interface{}
            json.Unmarshal([]byte(context.String), &contextData)
            log["context"] = contextData
        }
        
        logs = append(logs, log)
    }
    
    return logs, nil
}

func addLogEntry(agentID string, entry map[string]interface{}) error {
    query := `
        INSERT INTO agent_dashboard.agent_logs (agent_id, level, message, context)
        VALUES ($1, $2, $3, $4::jsonb)
    `
    
    level := entry["level"].(string)
    message := entry["message"].(string)
    context := entry["context"]
    
    _, err := db.Exec(query, agentID, level, message, mustMarshal(context))
    return err
}

func executeControl(agentID string, action string, params map[string]interface{}) (map[string]interface{}, error) {
    // Update agent status based on action
    var newStatus string
    switch action {
    case "start":
        newStatus = "starting"
    case "stop":
        newStatus = "stopping"
    case "restart":
        newStatus = "restarting"
    case "pause":
        newStatus = "paused"
    default:
        return nil, fmt.Errorf("unknown action: %s", action)
    }
    
    query := `UPDATE agent_dashboard.agents SET status = $1 WHERE id = $2`
    _, err := db.Exec(query, newStatus, agentID)
    if err != nil {
        return nil, err
    }
    
    // Publish control event to Redis
    controlEvent := map[string]interface{}{
        "agent_id":  agentID,
        "action":    action,
        "params":    params,
        "timestamp": time.Now(),
    }
    rdb.Publish(ctx, "agent_control", mustMarshal(controlEvent))
    
    return map[string]interface{}{
        "action":    action,
        "status":    newStatus,
        "timestamp": time.Now(),
    }, nil
}

func getSystemHealth() (map[string]interface{}, error) {
    query := `
        SELECT 
            COUNT(*) as total_agents,
            COUNT(CASE WHEN status = 'active' THEN 1 END) as active_agents,
            COUNT(CASE WHEN status = 'error' THEN 1 END) as error_agents,
            COUNT(CASE WHEN last_heartbeat > NOW() - INTERVAL '1 minute' THEN 1 END) as healthy_agents
        FROM agent_dashboard.agents
        WHERE status != 'terminated'
    `
    
    var total, active, errorCount, healthy int
    err := db.QueryRow(query).Scan(&total, &active, &errorCount, &healthy)
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "total_agents":   total,
        "active_agents":  active,
        "error_agents":   errorCount,
        "healthy_agents": healthy,
        "system_status":  determineSystemStatus(errorCount, active, healthy),
        "timestamp":      time.Now(),
    }, nil
}

func determineSystemStatus(errorCount, active, healthy int) string {
    if errorCount > 2 {
        return "critical"
    }
    if healthy < active/2 {
        return "warning"
    }
    return "healthy"
}

// Helper functions
func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, message string, status int) {
    response := APIResponse{
        Success: false,
        Error:   message,
    }
    jsonResponse(w, response, status)
}

func mustMarshal(v interface{}) string {
    data, _ := json.Marshal(v)
    return string(data)
}