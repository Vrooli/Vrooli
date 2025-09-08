package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "os"
    "strings"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/lib/pq"
    _ "github.com/lib/pq"
)

// MindMap represents a mind map structure
type MindMap struct {
    ID          string                 `json:"id"`
    Title       string                 `json:"title"`
    Description string                 `json:"description"`
    CampaignID  string                 `json:"campaign_id,omitempty"`
    OwnerID     string                 `json:"owner_id"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   string                 `json:"created_at"`
    UpdatedAt   string                 `json:"updated_at"`
}

// Node represents a mind map node
type Node struct {
    ID        string                 `json:"id"`
    MindMapID string                 `json:"mind_map_id"`
    Content   string                 `json:"content"`
    Type      string                 `json:"type"`
    PositionX float64                `json:"position_x"`
    PositionY float64                `json:"position_y"`
    ParentID  *string                `json:"parent_id,omitempty"`
    Metadata  map[string]interface{} `json:"metadata"`
}

// Response represents a standard API response
type Response struct {
    Status  string      `json:"status"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}

// Global database connection and processor
var db *sql.DB
var mindMapProcessor *MindMapProcessor

// Initialize database connection and schema
func initDB() error {
    // PostgreSQL URL must come from environment - no defaults
    postgresURL := os.Getenv("POSTGRES_URL")
    if postgresURL == "" {
        // Try building from individual components if URL not provided
        host := os.Getenv("POSTGRES_HOST")
        port := os.Getenv("POSTGRES_PORT")
        user := os.Getenv("POSTGRES_USER")
        password := os.Getenv("POSTGRES_PASSWORD")
        dbname := os.Getenv("POSTGRES_DB")
        
        if host == "" || port == "" || user == "" || password == "" || dbname == "" {
            return fmt.Errorf("database configuration required: provide POSTGRES_URL or all of POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
        }
        
        postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
            user, password, host, port, dbname)
    }
    
    log.Printf("🔄 Connecting to database (credentials hidden)")
    
    var err error
    db, err = sql.Open("postgres", postgresURL)
    if err != nil {
        return fmt.Errorf("failed to open database connection: %v", err)
    }
    
    // Configure connection pool for resilience
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Implement exponential backoff for database connection
    maxRetries := 10
    baseDelay := 1 * time.Second
    maxDelay := 30 * time.Second
    
    log.Println("🔄 Attempting database connection with exponential backoff...")
    
    var pingErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        pingErr = db.Ping()
        if pingErr == nil {
            log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
            break
        }
        
        // Calculate exponential backoff delay with cap
        delay := time.Duration(math.Min(
            float64(baseDelay) * math.Pow(2, float64(attempt)),
            float64(maxDelay),
        ))
        
        // Add jitter to prevent thundering herd (0-25% additional delay)
        jitterFactor := math.Min(float64(attempt)/float64(maxRetries), 0.25)
        jitter := time.Duration(float64(delay) * jitterFactor)
        actualDelay := delay + jitter
        
        log.Printf("⚠️  Database connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
        log.Printf("⏳ Waiting %v before retry (base: %v, jitter: %v)", actualDelay, delay, jitter)
        
        // Show progress stats every few attempts
        if attempt > 0 && attempt % 3 == 0 {
            log.Printf("📊 Connection retry statistics:")
            log.Printf("   - Attempts: %d of %d", attempt + 1, maxRetries)
            log.Printf("   - Time elapsed: ~%v", time.Duration(attempt * 2) * baseDelay)
            log.Printf("   - Max wait time: %v", maxDelay)
        }
        
        time.Sleep(actualDelay)
    }
    
    if pingErr != nil {
        return fmt.Errorf("database connection failed after %d attempts: %v", maxRetries, pingErr)
    }
    
    log.Println("🎉 Database connection pool established successfully!")
    
    // Check if proper schema exists
    var exists bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'mind_maps')").Scan(&exists)
    if err != nil {
        return fmt.Errorf("failed to check schema: %v", err)
    }
    
    if exists {
        log.Println("✅ Using existing PostgreSQL schema with full mind maps features")
        return nil
    }
    
    log.Println("⚠️ Mind maps schema not found - creating minimal tables for basic functionality")
    log.Println("💡 For full features, run the PostgreSQL schema from initialization/postgres/schema.sql")
    
    // Create minimal tables if proper schema doesn't exist
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS mind_maps (
            id SERIAL PRIMARY KEY,
            title VARCHAR(255) NOT NULL,
            description TEXT,
            campaign_id VARCHAR(255),
            owner_id VARCHAR(255) DEFAULT 'default-user',
            metadata JSONB DEFAULT '{}',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        CREATE TABLE IF NOT EXISTS mind_map_nodes (
            id SERIAL PRIMARY KEY,
            mind_map_id INTEGER REFERENCES mind_maps(id) ON DELETE CASCADE,
            content TEXT NOT NULL,
            type VARCHAR(50) DEFAULT 'child',
            position_x FLOAT DEFAULT 0,
            position_y FLOAT DEFAULT 0,
            parent_id INTEGER REFERENCES mind_map_nodes(id),
            metadata JSONB DEFAULT '{}',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `)
    if err != nil {
        return fmt.Errorf("failed to create minimal schema: %v", err)
    }
    
    log.Println("✅ Minimal mind maps schema created successfully")
    return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := Response{
        Status:  "success",
        Message: "Mind Maps API is running",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getMindMapsHandler(w http.ResponseWriter, r *http.Request) {
    ownerID := r.URL.Query().Get("owner_id")
    if ownerID == "" {
        ownerID = "default-user"
    }
    
    query := `
        SELECT id, title, description, COALESCE(campaign_id, ''), owner_id, 
               COALESCE(metadata, '{}'), created_at, updated_at
        FROM mind_maps 
        WHERE owner_id = $1 
        ORDER BY updated_at DESC
    `
    
    rows, err := db.Query(query, ownerID)
    if err != nil {
        log.Printf("Error querying mind maps: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var mindMaps []MindMap
    for rows.Next() {
        var m MindMap
        var metadataBytes []byte
        var createdAt, updatedAt time.Time
        
        err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.CampaignID, 
                        &m.OwnerID, &metadataBytes, &createdAt, &updatedAt)
        if err != nil {
            log.Printf("Error scanning mind map: %v", err)
            continue
        }
        
        // Parse metadata JSON
        if len(metadataBytes) > 0 {
            json.Unmarshal(metadataBytes, &m.Metadata)
        } else {
            m.Metadata = make(map[string]interface{})
        }
        
        m.CreatedAt = createdAt.Format(time.RFC3339)
        m.UpdatedAt = updatedAt.Format(time.RFC3339)
        
        mindMaps = append(mindMaps, m)
    }
    
    if mindMaps == nil {
        mindMaps = []MindMap{}
    }
    
    response := Response{
        Status: "success",
        Data:   mindMaps,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func createMindMapHandler(w http.ResponseWriter, r *http.Request) {
    var mindMap MindMap
    if err := json.NewDecoder(r.Body).Decode(&mindMap); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Set defaults
    if mindMap.OwnerID == "" {
        mindMap.OwnerID = "default-user"
    }
    if mindMap.Metadata == nil {
        mindMap.Metadata = make(map[string]interface{})
    }
    
    // Convert metadata to JSON
    metadataBytes, err := json.Marshal(mindMap.Metadata)
    if err != nil {
        log.Printf("Error marshaling metadata: %v", err)
        http.Error(w, "Invalid metadata", http.StatusBadRequest)
        return
    }
    
    query := `
        INSERT INTO mind_maps (title, description, campaign_id, owner_id, metadata, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        RETURNING id, created_at, updated_at
    `
    
    var id string
    var createdAt, updatedAt time.Time
    err = db.QueryRow(query, mindMap.Title, mindMap.Description, 
                     mindMap.CampaignID, mindMap.OwnerID, metadataBytes).
                     Scan(&id, &createdAt, &updatedAt)
    if err != nil {
        log.Printf("Error creating mind map: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    mindMap.ID = id
    mindMap.CreatedAt = createdAt.Format(time.RFC3339)
    mindMap.UpdatedAt = updatedAt.Format(time.RFC3339)
    
    // Trigger auto-organization in background
    go func() {
        ctx := context.WithTimeout(context.Background(), 30*time.Second)
        defer ctx.Done()
        
        req := OrganizeRequest{
            MindMapID: id,
            Method:    "basic",
        }
        
        if err := mindMapProcessor.AutoOrganize(ctx, req); err != nil {
            log.Printf("⚠️ Auto-organize failed (non-critical): %v", err)
        } else {
            log.Printf("✅ Auto-organize completed for mind map: %s", id)
        }
    }()
    
    response := Response{
        Status:  "success",
        Message: "Mind map created successfully",
        Data:    mindMap,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getMindMapHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    query := `
        SELECT id, title, description, COALESCE(campaign_id, ''), owner_id, 
               COALESCE(metadata, '{}'), created_at, updated_at
        FROM mind_maps 
        WHERE id = $1
    `
    
    var m MindMap
    var metadataBytes []byte
    var createdAt, updatedAt time.Time
    
    err := db.QueryRow(query, id).Scan(&m.ID, &m.Title, &m.Description, 
                                      &m.CampaignID, &m.OwnerID, &metadataBytes, 
                                      &createdAt, &updatedAt)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Mind map not found", http.StatusNotFound)
            return
        }
        log.Printf("Error querying mind map: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    // Parse metadata JSON
    if len(metadataBytes) > 0 {
        json.Unmarshal(metadataBytes, &m.Metadata)
    } else {
        m.Metadata = make(map[string]interface{})
    }
    
    m.CreatedAt = createdAt.Format(time.RFC3339)
    m.UpdatedAt = updatedAt.Format(time.RFC3339)
    
    response := Response{
        Status: "success",
        Data:   m,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func updateMindMapHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    var updates map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Build dynamic update query
    setParts := []string{}
    args := []interface{}{}
    argIndex := 1
    
    if title, ok := updates["title"].(string); ok {
        setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
        args = append(args, title)
        argIndex++
    }
    
    if description, ok := updates["description"].(string); ok {
        setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
        args = append(args, description)
        argIndex++
    }
    
    if campaignID, ok := updates["campaign_id"].(string); ok {
        setParts = append(setParts, fmt.Sprintf("campaign_id = $%d", argIndex))
        args = append(args, campaignID)
        argIndex++
    }
    
    if metadata, ok := updates["metadata"]; ok {
        metadataBytes, err := json.Marshal(metadata)
        if err != nil {
            http.Error(w, "Invalid metadata format", http.StatusBadRequest)
            return
        }
        setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
        args = append(args, metadataBytes)
        argIndex++
    }
    
    if len(setParts) == 0 {
        http.Error(w, "No valid fields to update", http.StatusBadRequest)
        return
    }
    
    // Add updated_at and id
    setParts = append(setParts, fmt.Sprintf("updated_at = CURRENT_TIMESTAMP"))
    args = append(args, id)
    
    query := fmt.Sprintf("UPDATE mind_maps SET %s WHERE id = $%d", 
                        strings.Join(setParts, ", "), argIndex)
    
    result, err := db.Exec(query, args...)
    if err != nil {
        log.Printf("Error updating mind map: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Mind map not found", http.StatusNotFound)
        return
    }
    
    response := Response{
        Status:  "success",
        Message: fmt.Sprintf("Mind map %s updated successfully", id),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func deleteMindMapHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    query := "DELETE FROM mind_maps WHERE id = $1"
    result, err := db.Exec(query, id)
    if err != nil {
        log.Printf("Error deleting mind map: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Mind map not found", http.StatusNotFound)
        return
    }
    
    response := Response{
        Status:  "success",
        Message: fmt.Sprintf("Mind map %s deleted successfully", id),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func addNodeHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    mindMapID := vars["id"]
    
    var node Node
    if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Set defaults
    node.MindMapID = mindMapID
    if node.Type == "" {
        node.Type = "child"
    }
    if node.Metadata == nil {
        node.Metadata = make(map[string]interface{})
    }
    
    // Convert metadata to JSON
    metadataBytes, err := json.Marshal(node.Metadata)
    if err != nil {
        log.Printf("Error marshaling node metadata: %v", err)
        http.Error(w, "Invalid metadata", http.StatusBadRequest)
        return
    }
    
    query := `
        INSERT INTO mind_map_nodes (mind_map_id, content, type, position_x, position_y, parent_id, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `
    
    var nodeID string
    err = db.QueryRow(query, mindMapID, node.Content, node.Type, 
                     node.PositionX, node.PositionY, node.ParentID, metadataBytes).
                     Scan(&nodeID)
    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
            http.Error(w, "Mind map not found or invalid parent node", http.StatusBadRequest)
            return
        }
        log.Printf("Error creating node: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    node.ID = nodeID
    
    response := Response{
        Status:  "success",
        Message: "Node added successfully",
        Data:    node,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getNodesHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    mindMapID := vars["id"]
    
    query := `
        SELECT id, mind_map_id, content, type, position_x, position_y, parent_id, metadata
        FROM mind_map_nodes 
        WHERE mind_map_id = $1 
        ORDER BY id
    `
    
    rows, err := db.Query(query, mindMapID)
    if err != nil {
        log.Printf("Error querying nodes: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var nodes []Node
    for rows.Next() {
        var n Node
        var metadataBytes []byte
        var parentID sql.NullString
        
        err := rows.Scan(&n.ID, &n.MindMapID, &n.Content, &n.Type, 
                        &n.PositionX, &n.PositionY, &parentID, &metadataBytes)
        if err != nil {
            log.Printf("Error scanning node: %v", err)
            continue
        }
        
        if parentID.Valid {
            n.ParentID = &parentID.String
        }
        
        // Parse metadata JSON
        if len(metadataBytes) > 0 {
            json.Unmarshal(metadataBytes, &n.Metadata)
        } else {
            n.Metadata = make(map[string]interface{})
        }
        
        nodes = append(nodes, n)
    }
    
    if nodes == nil {
        nodes = []Node{}
    }
    
    response := Response{
        Status: "success",
        Data:   nodes,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getNodeHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    nodeID := vars["nodeId"]
    mapID := vars["mapId"]
    
    query := `
        SELECT id, mind_map_id, content, type, position_x, position_y, parent_id, metadata
        FROM mind_map_nodes 
        WHERE id = $1 AND mind_map_id = $2
    `
    
    var n Node
    var metadataBytes []byte
    var parentID sql.NullString
    
    err := db.QueryRow(query, nodeID, mapID).Scan(&n.ID, &n.MindMapID, &n.Content, 
                                                  &n.Type, &n.PositionX, &n.PositionY, 
                                                  &parentID, &metadataBytes)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Node not found", http.StatusNotFound)
            return
        }
        log.Printf("Error querying node: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    if parentID.Valid {
        n.ParentID = &parentID.String
    }
    
    // Parse metadata JSON
    if len(metadataBytes) > 0 {
        json.Unmarshal(metadataBytes, &n.Metadata)
    } else {
        n.Metadata = make(map[string]interface{})
    }
    
    response := Response{
        Status: "success",
        Data:   n,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func updateNodeHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    nodeID := vars["nodeId"]
    mapID := vars["mapId"]
    
    var updates map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Build dynamic update query
    setParts := []string{}
    args := []interface{}{}
    argIndex := 1
    
    if content, ok := updates["content"].(string); ok {
        setParts = append(setParts, fmt.Sprintf("content = $%d", argIndex))
        args = append(args, content)
        argIndex++
    }
    
    if nodeType, ok := updates["type"].(string); ok {
        setParts = append(setParts, fmt.Sprintf("type = $%d", argIndex))
        args = append(args, nodeType)
        argIndex++
    }
    
    if posX, ok := updates["position_x"].(float64); ok {
        setParts = append(setParts, fmt.Sprintf("position_x = $%d", argIndex))
        args = append(args, posX)
        argIndex++
    }
    
    if posY, ok := updates["position_y"].(float64); ok {
        setParts = append(setParts, fmt.Sprintf("position_y = $%d", argIndex))
        args = append(args, posY)
        argIndex++
    }
    
    if parentID, ok := updates["parent_id"]; ok {
        setParts = append(setParts, fmt.Sprintf("parent_id = $%d", argIndex))
        args = append(args, parentID)
        argIndex++
    }
    
    if metadata, ok := updates["metadata"]; ok {
        metadataBytes, err := json.Marshal(metadata)
        if err != nil {
            http.Error(w, "Invalid metadata format", http.StatusBadRequest)
            return
        }
        setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
        args = append(args, metadataBytes)
        argIndex++
    }
    
    if len(setParts) == 0 {
        http.Error(w, "No valid fields to update", http.StatusBadRequest)
        return
    }
    
    // Add updated_at and conditions
    setParts = append(setParts, "updated_at = CURRENT_TIMESTAMP")
    args = append(args, nodeID, mapID)
    
    query := fmt.Sprintf("UPDATE mind_map_nodes SET %s WHERE id = $%d AND mind_map_id = $%d", 
                        strings.Join(setParts, ", "), argIndex, argIndex+1)
    
    result, err := db.Exec(query, args...)
    if err != nil {
        log.Printf("Error updating node: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Node not found", http.StatusNotFound)
        return
    }
    
    response := Response{
        Status:  "success",
        Message: fmt.Sprintf("Node %s updated successfully", nodeID),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func deleteNodeHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    nodeID := vars["nodeId"]
    mapID := vars["mapId"]
    
    query := "DELETE FROM mind_map_nodes WHERE id = $1 AND mind_map_id = $2"
    result, err := db.Exec(query, nodeID, mapID)
    if err != nil {
        log.Printf("Error deleting node: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        http.Error(w, "Node not found", http.StatusNotFound)
        return
    }
    
    response := Response{
        Status:  "success",
        Message: fmt.Sprintf("Node %s deleted successfully", nodeID),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
    var searchRequest map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    query, ok := searchRequest["query"].(string)
    if !ok || query == "" {
        http.Error(w, "Query string required", http.StatusBadRequest)
        return
    }
    
    ownerID := "default-user"
    if owner, ok := searchRequest["owner_id"].(string); ok {
        ownerID = owner
    }
    
    searchMode := "simple"
    if mode, ok := searchRequest["search_mode"].(string); ok {
        searchMode = mode
    }
    
    // Try semantic search if requested
    if searchMode == "semantic" {
        log.Printf("Attempting semantic search for query: %s", query)
        
        ctx := context.WithTimeout(context.Background(), 10*time.Second)
        searchReq := SemanticSearchRequest{
            Query:      query,
            Collection: "mind_maps",
            Limit:      20,
        }
        
        if results, err := mindMapProcessor.SemanticSearch(ctx, searchReq); err == nil {
            log.Println("✅ Semantic search successful")
            response := Response{
                Status: "success",
                Data: map[string]interface{}{
                    "results":     results,
                    "query":       query,
                    "search_mode": "semantic",
                    "source":      "qdrant",
                },
            }
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(response)
            return
        } else {
            log.Printf("⚠️ Semantic search failed, falling back to simple search: %v", err)
        }
    }
    
    // Search across mind maps and nodes using full-text search
    searchQuery := `
        SELECT DISTINCT m.id, m.title, m.description, COALESCE(m.campaign_id, ''), 
               m.owner_id, COALESCE(m.metadata, '{}'), m.created_at, m.updated_at,
               'mindmap' as result_type, '' as node_content
        FROM mind_maps m 
        WHERE m.owner_id = $1 AND (
            m.title ILIKE $2 OR 
            m.description ILIKE $2
        )
        UNION
        SELECT DISTINCT m.id, m.title, m.description, COALESCE(m.campaign_id, ''), 
               m.owner_id, COALESCE(m.metadata, '{}'), m.created_at, m.updated_at,
               'node' as result_type, n.content as node_content
        FROM mind_maps m 
        JOIN mind_map_nodes n ON m.id = n.mind_map_id
        WHERE m.owner_id = $1 AND n.content ILIKE $2
        ORDER BY created_at DESC
        LIMIT 20
    `
    
    searchPattern := "%" + query + "%"
    rows, err := db.Query(searchQuery, ownerID, searchPattern)
    if err != nil {
        log.Printf("Error searching: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var results []map[string]interface{}
    for rows.Next() {
        var m MindMap
        var metadataBytes []byte
        var createdAt, updatedAt time.Time
        var resultType, nodeContent string
        
        err := rows.Scan(&m.ID, &m.Title, &m.Description, &m.CampaignID, 
                        &m.OwnerID, &metadataBytes, &createdAt, &updatedAt,
                        &resultType, &nodeContent)
        if err != nil {
            log.Printf("Error scanning search result: %v", err)
            continue
        }
        
        // Parse metadata JSON
        if len(metadataBytes) > 0 {
            json.Unmarshal(metadataBytes, &m.Metadata)
        } else {
            m.Metadata = make(map[string]interface{})
        }
        
        m.CreatedAt = createdAt.Format(time.RFC3339)
        m.UpdatedAt = updatedAt.Format(time.RFC3339)
        
        result := map[string]interface{}{
            "mindmap":     m,
            "type":        resultType,
            "match_text":  nodeContent,
        }
        
        results = append(results, result)
    }
    
    if results == nil {
        results = []map[string]interface{}{}
    }
    
    response := Response{
        Status: "success",
        Data: map[string]interface{}{
            "results":     results,
            "query":       query,
            "count":       len(results),
            "search_mode": "simple",
            "source":      "database",
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    format := r.URL.Query().Get("format")
    if format == "" {
        format = "json"
    }
    
    response := Response{
        Status: "success",
        Data: map[string]interface{}{
            "mind_map_id": id,
            "format":      format,
            "url":         fmt.Sprintf("/exports/%s.%s", id, format),
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func enableCORS(next http.Handler) http.Handler {
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

func main() {
    // Initialize database connection
    if err := initDB(); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close()
    
    // Initialize mind map processor
    ollamaURL := os.Getenv("OLLAMA_URL")
    if ollamaURL == "" {
        ollamaURL = "http://localhost:11434"
    }
    
    qdrantURL := os.Getenv("QDRANT_URL")
    if qdrantURL == "" {
        qdrantURL = "http://localhost:6333"
    }
    
    mindMapProcessor = NewMindMapProcessor(db, ollamaURL, qdrantURL)
    log.Printf("Initialized MindMapProcessor with Ollama: %s, Qdrant: %s", ollamaURL, qdrantURL)
    
    r := mux.NewRouter()
    
    // Health check
    r.HandleFunc("/health", healthHandler).Methods("GET")
    
    // Mind map endpoints
    r.HandleFunc("/api/mindmaps", getMindMapsHandler).Methods("GET")
    r.HandleFunc("/api/mindmaps", createMindMapHandler).Methods("POST")
    r.HandleFunc("/api/mindmaps/{id}", getMindMapHandler).Methods("GET")
    r.HandleFunc("/api/mindmaps/{id}", updateMindMapHandler).Methods("PUT")
    r.HandleFunc("/api/mindmaps/{id}", deleteMindMapHandler).Methods("DELETE")
    
    // Node endpoints
    r.HandleFunc("/api/mindmaps/{id}/nodes", getNodesHandler).Methods("GET")
    r.HandleFunc("/api/mindmaps/{id}/nodes", addNodeHandler).Methods("POST")
    r.HandleFunc("/api/mindmaps/{mapId}/nodes/{nodeId}", getNodeHandler).Methods("GET")
    r.HandleFunc("/api/mindmaps/{mapId}/nodes/{nodeId}", updateNodeHandler).Methods("PUT")
    r.HandleFunc("/api/mindmaps/{mapId}/nodes/{nodeId}", deleteNodeHandler).Methods("DELETE")
    
    // Search endpoint
    r.HandleFunc("/api/mindmaps/search", searchHandler).Methods("POST")
    
    // Export endpoint
    r.HandleFunc("/api/mindmaps/{id}/export", exportHandler).Methods("GET")
    
    // Apply CORS middleware
    handler := enableCORS(r)
    
    // API port is required from environment - no defaults
    port := os.Getenv("API_PORT")
    if port == "" {
        port = os.Getenv("PORT")
    }
    if port == "" {
        log.Fatal("❌ Missing required API_PORT or PORT environment variable")
    }
    
    log.Printf("🚀 Mind Maps API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        log.Fatal(err)
    }
}
