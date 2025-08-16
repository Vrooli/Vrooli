package main

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    _ "github.com/lib/pq"
    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

type Score struct {
    ID         int       `json:"id"`
    Name       string    `json:"name"`
    Score      int       `json:"score"`
    WPM        int       `json:"wpm"`
    Accuracy   int       `json:"accuracy"`
    MaxCombo   int       `json:"maxCombo"`
    Difficulty string    `json:"difficulty"`
    Mode       string    `json:"mode"`
    CreatedAt  time.Time `json:"createdAt"`
}

type LeaderboardResponse struct {
    Scores []Score `json:"scores"`
    Period string  `json:"period"`
}

type StatsRequest struct {
    SessionID string `json:"sessionId"`
    WPM       int    `json:"wpm"`
    Accuracy  int    `json:"accuracy"`
    Text      string `json:"text"`
}

type CoachingRequest struct {
    UserStats  StatsRequest `json:"userStats"`
    Difficulty string       `json:"difficulty"`
}

var db *sql.DB
var n8nURL string

func main() {
    // Get environment variables
    port := os.Getenv("PORT")
    if port == "" {
        port = "9200"
    }

    postgresURL := os.Getenv("POSTGRES_URL")
    if postgresURL == "" {
        postgresURL = "postgres://postgres:postgres@localhost:5432/typing_test?sslmode=disable"
    }

    n8nURL = os.Getenv("N8N_BASE_URL")
    if n8nURL == "" {
        n8nURL = "http://localhost:5678"
    }

    // Connect to database
    var err error
    db, err = sql.Open("postgres", postgresURL)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()

    // Initialize database
    initDB()

    // Setup routes
    router := mux.NewRouter()
    
    // API routes
    router.HandleFunc("/health", healthHandler).Methods("GET")
    router.HandleFunc("/api/leaderboard", getLeaderboard).Methods("GET")
    router.HandleFunc("/api/submit-score", submitScore).Methods("POST")
    router.HandleFunc("/api/stats", submitStats).Methods("POST")
    router.HandleFunc("/api/coaching", getCoaching).Methods("POST")
    router.HandleFunc("/api/practice-text", getPracticeText).Methods("GET")

    // Setup CORS
    c := cors.New(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"*"},
    })

    handler := c.Handler(router)

    log.Printf("🎮 Typing Test API running on port %s", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

func initDB() {
    query := `
    CREATE TABLE IF NOT EXISTS scores (
        id SERIAL PRIMARY KEY,
        name VARCHAR(50) NOT NULL,
        score INTEGER NOT NULL,
        wpm INTEGER NOT NULL,
        accuracy INTEGER NOT NULL,
        max_combo INTEGER DEFAULT 0,
        difficulty VARCHAR(20) DEFAULT 'easy',
        mode VARCHAR(20) DEFAULT 'classic',
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS typing_sessions (
        id SERIAL PRIMARY KEY,
        session_id VARCHAR(100) UNIQUE NOT NULL,
        total_chars INTEGER DEFAULT 0,
        correct_chars INTEGER DEFAULT 0,
        total_time INTEGER DEFAULT 0,
        texts_completed INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_scores_created_at ON scores(created_at DESC);
    CREATE INDEX IF NOT EXISTS idx_scores_score ON scores(score DESC);
    CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON typing_sessions(session_id);
    `

    if _, err := db.Exec(query); err != nil {
        log.Printf("Warning: Failed to initialize database: %v", err)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getLeaderboard(w http.ResponseWriter, r *http.Request) {
    period := r.URL.Query().Get("period")
    if period == "" {
        period = "all"
    }

    var whereClause string
    switch period {
    case "today":
        whereClause = "WHERE created_at >= CURRENT_DATE"
    case "week":
        whereClause = "WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'"
    case "month":
        whereClause = "WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'"
    default:
        whereClause = ""
    }

    query := fmt.Sprintf(`
        SELECT id, name, score, wpm, accuracy, max_combo, difficulty, mode, created_at
        FROM scores
        %s
        ORDER BY score DESC
        LIMIT 10
    `, whereClause)

    rows, err := db.Query(query)
    if err != nil {
        http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var scores []Score
    for rows.Next() {
        var score Score
        err := rows.Scan(&score.ID, &score.Name, &score.Score, &score.WPM, 
            &score.Accuracy, &score.MaxCombo, &score.Difficulty, &score.Mode, &score.CreatedAt)
        if err != nil {
            continue
        }
        scores = append(scores, score)
    }

    response := LeaderboardResponse{
        Scores: scores,
        Period: period,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func submitScore(w http.ResponseWriter, r *http.Request) {
    var score Score
    if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Insert score into database
    query := `
        INSERT INTO scores (name, score, wpm, accuracy, max_combo, difficulty, mode)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at
    `

    err := db.QueryRow(query, score.Name, score.Score, score.WPM, 
        score.Accuracy, score.MaxCombo, score.Difficulty, score.Mode).Scan(&score.ID, &score.CreatedAt)
    if err != nil {
        http.Error(w, "Failed to save score", http.StatusInternalServerError)
        return
    }

    // Trigger n8n workflow for leaderboard update
    go triggerLeaderboardUpdate(score)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "scoreId": score.ID,
    })
}

func submitStats(w http.ResponseWriter, r *http.Request) {
    var stats StatsRequest
    if err := json.NewDecoder(r.Body).Decode(&stats); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Update or create session stats
    query := `
        INSERT INTO typing_sessions (session_id, total_chars, correct_chars, total_time, texts_completed)
        VALUES ($1, $2, $3, $4, 1)
        ON CONFLICT (session_id) 
        DO UPDATE SET 
            total_chars = typing_sessions.total_chars + EXCLUDED.total_chars,
            correct_chars = typing_sessions.correct_chars + EXCLUDED.correct_chars,
            total_time = typing_sessions.total_time + EXCLUDED.total_time,
            texts_completed = typing_sessions.texts_completed + 1,
            updated_at = CURRENT_TIMESTAMP
    `

    totalChars := len(stats.Text)
    correctChars := int(float64(totalChars) * float64(stats.Accuracy) / 100)
    timeSeconds := 60 // Default session time

    _, err := db.Exec(query, stats.SessionID, totalChars, correctChars, timeSeconds)
    if err != nil {
        log.Printf("Failed to update stats: %v", err)
    }

    // Trigger stats processing workflow
    go triggerStatsProcessing(stats)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func getCoaching(w http.ResponseWriter, r *http.Request) {
    var request CoachingRequest
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get session history from database
    var avgWPM, avgAccuracy, totalSessions int
    var bestWPM int
    query := `
        SELECT 
            COALESCE(AVG(wpm), 0)::INT as avg_wpm,
            COALESCE(AVG(accuracy), 0)::INT as avg_accuracy,
            COUNT(*) as total_sessions,
            COALESCE(MAX(wpm), 0) as best_wpm
        FROM scores
        WHERE created_at > NOW() - INTERVAL '30 days'
    `
    db.QueryRow(query).Scan(&avgWPM, &avgAccuracy, &totalSessions, &bestWPM)

    // Determine trend
    trend := "stable"
    if request.UserStats.WPM > avgWPM+5 {
        trend = "improving"
    } else if request.UserStats.WPM < avgWPM-5 {
        trend = "declining"
    }

    // Call n8n coaching workflow if available
    if n8nURL != "" {
        payload := map[string]interface{}{
            "userId": fmt.Sprintf("user_%d", time.Now().Unix()),
            "sessionStats": map[string]interface{}{
                "wpm":         request.UserStats.WPM,
                "accuracy":    request.UserStats.Accuracy,
                "improvement": request.UserStats.WPM - avgWPM,
            },
            "historicalData": map[string]interface{}{
                "avgWpm":        avgWPM,
                "avgAccuracy":   avgAccuracy,
                "totalSessions": totalSessions,
                "bestWpm":       bestWPM,
                "trend":         trend,
            },
        }

        jsonData, _ := json.Marshal(payload)
        webhookURL := fmt.Sprintf("%s/webhook/typing-test/coach", n8nURL)
        
        resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
        if err == nil && resp.StatusCode == 200 {
            defer resp.Body.Close()
            var result map[string]interface{}
            if json.NewDecoder(resp.Body).Decode(&result) == nil {
                w.Header().Set("Content-Type", "application/json")
                json.NewEncoder(w).Encode(result)
                return
            }
        }
    }

    // Fallback to enhanced static tips based on performance
    tips := map[string][]string{
        "beginner": {
            "Start with home row keys (ASDF JKL;) - place your fingers there now",
            "Type slowly but accurately - speed comes with muscle memory",
            "Look at the screen, not the keyboard - trust your fingers",
        },
        "intermediate": {
            "Focus on your weak keys - practice words with those letters",
            "Try typing common word patterns like 'ing', 'tion', 'the'",
            "Maintain rhythm - consistent timing improves flow",
        },
        "advanced": {
            "Challenge yourself with code or technical documentation",
            "Practice typing while thinking about content, not keys",
            "Try speed bursts - type as fast as possible for 10 seconds",
        },
    }

    skillLevel := "beginner"
    if request.UserStats.WPM >= 60 && request.UserStats.Accuracy >= 95 {
        skillLevel = "advanced"
    } else if request.UserStats.WPM >= 40 && request.UserStats.Accuracy >= 90 {
        skillLevel = "intermediate"
    }

    selectedTips := tips[skillLevel]
    selectedTip := selectedTips[request.UserStats.WPM%len(selectedTips)]

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "coaching": map[string]interface{}{
            "technique": selectedTip,
            "encouragement": fmt.Sprintf("You're typing at %d WPM! Keep practicing!", request.UserStats.WPM),
            "personalizedMetrics": map[string]interface{}{
                "currentWpm": request.UserStats.WPM,
                "targetWpm": request.UserStats.WPM + 10,
                "accuracyGoal": min(request.UserStats.Accuracy + 5, 100),
            },
        },
        "recommendedDifficulty": getRecommendedDifficulty(request.UserStats.WPM, request.UserStats.Accuracy),
    })
}

func getPracticeText(w http.ResponseWriter, r *http.Request) {
    difficulty := r.URL.Query().Get("difficulty")
    if difficulty == "" {
        difficulty = "easy"
    }

    // In production, this would fetch from database or n8n workflow
    texts := map[string][]string{
        "easy": {
            "The cat sat on the mat and looked at the moon",
            "A bird in hand is worth two in the bush today",
            "Time flies when you are having fun with friends",
        },
        "medium": {
            "The extraordinary symphony echoed through the vast cathedral halls magnificently",
            "Quantum mechanics revolutionizes our understanding of subatomic particle behavior",
            "Artificial intelligence transforms industries through automated decision-making processes",
        },
        "hard": {
            "Pseudopseudohypoparathyroidism represents complex endocrinological dysfunction requiring specialized treatment",
            "Cryptographically secure pseudorandom generators utilize mathematical algorithms ensuring unpredictability",
            "Neuroplasticity demonstrates the brain's remarkable capacity for reorganization throughout life",
        },
    }

    selectedTexts := texts[difficulty]
    if selectedTexts == nil {
        selectedTexts = texts["easy"]
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "texts": selectedTexts,
        "difficulty": difficulty,
    })
}

func getRecommendedDifficulty(wpm, accuracy int) string {
    if wpm > 60 && accuracy > 95 {
        return "hard"
    } else if wpm > 40 && accuracy > 90 {
        return "medium"
    }
    return "easy"
}

func triggerLeaderboardUpdate(score Score) {
    // This would trigger the n8n leaderboard-manager workflow
    // For now, just log it
    log.Printf("New score submitted: %s - %d points", score.Name, score.Score)
}

func triggerStatsProcessing(stats StatsRequest) {
    // This would trigger the n8n typing-stats-processor workflow
    // For now, just log it
    log.Printf("Stats processed for session: %s", stats.SessionID)
}