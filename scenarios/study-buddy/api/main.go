package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
)

var db *sql.DB

type StudyMaterial struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    SubjectID   string    `json:"subject_id"`
    Title       string    `json:"title"`
    Content     string    `json:"content"`
    MaterialType string   `json:"material_type"`
    Tags        []string  `json:"tags"`
    CreatedAt   time.Time `json:"created_at"`
}

type Flashcard struct {
    ID         string    `json:"id"`
    UserID     string    `json:"user_id"`
    SubjectID  string    `json:"subject_id"`
    Front      string    `json:"front"`
    Back       string    `json:"back"`
    Difficulty int       `json:"difficulty"`
    Tags       []string  `json:"tags"`
    CreatedAt  time.Time `json:"created_at"`
}

type StudySession struct {
    ID               string    `json:"id"`
    UserID           string    `json:"user_id"`
    SubjectID        string    `json:"subject_id"`
    SessionType      string    `json:"session_type"`
    StartedAt        time.Time `json:"started_at"`
    EndedAt          *time.Time `json:"ended_at"`
    CardsReviewed    int       `json:"cards_reviewed"`
    QuestionsAnswered int      `json:"questions_answered"`
    CorrectAnswers   int       `json:"correct_answers"`
}

type Subject struct {
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Color       string    `json:"color"`
    Icon        string    `json:"icon"`
    CreatedAt   time.Time `json:"created_at"`
}

func initDB() {
    godotenv.Load()
    
    dbHost := getEnv("POSTGRES_HOST", "localhost")
    dbPort := getEnv("POSTGRES_PORT", "5432")
    dbUser := getEnv("POSTGRES_USER", "postgres")
    dbPassword := getEnv("POSTGRES_PASSWORD", "postgres")
    dbName := getEnv("POSTGRES_DB", "study_buddy")
    
    psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        dbHost, dbPort, dbUser, dbPassword, dbName)
    
    var err error
    db, err = sql.Open("postgres", psqlInfo)
    if err != nil {
        log.Fatal(err)
    }
    
    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Successfully connected to database")
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func main() {
    initDB()
    defer db.Close()
    
    router := gin.Default()
    router.Use(cors.Default())
    
    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy", "service": "study-buddy"})
    })
    
    // Subject endpoints
    router.POST("/api/subjects", createSubject)
    router.GET("/api/subjects/:userId", getUserSubjects)
    router.PUT("/api/subjects/:id", updateSubject)
    router.DELETE("/api/subjects/:id", deleteSubject)
    
    // Study material endpoints
    router.POST("/api/materials", createStudyMaterial)
    router.GET("/api/materials/:userId", getUserMaterials)
    router.GET("/api/materials/:userId/:subjectId", getSubjectMaterials)
    router.PUT("/api/materials/:id", updateStudyMaterial)
    router.DELETE("/api/materials/:id", deleteStudyMaterial)
    
    // Flashcard endpoints
    router.POST("/api/flashcards", createFlashcard)
    router.GET("/api/flashcards/:userId", getUserFlashcards)
    router.GET("/api/flashcards/:userId/:subjectId", getSubjectFlashcards)
    router.GET("/api/flashcards/due/:userId", getDueFlashcards)
    router.POST("/api/flashcards/:id/review", reviewFlashcard)
    
    // Study session endpoints
    router.POST("/api/sessions", startStudySession)
    router.PUT("/api/sessions/:id/end", endStudySession)
    router.GET("/api/sessions/:userId", getUserSessions)
    router.GET("/api/sessions/:userId/stats", getStudyStats)
    
    // Quiz endpoints
    router.POST("/api/quiz/generate", generateQuiz)
    router.POST("/api/quiz/submit", submitQuizAnswers)
    router.GET("/api/quiz/:userId/history", getQuizHistory)
    
    // Learning analytics
    router.GET("/api/analytics/:userId", getLearningAnalytics)
    router.GET("/api/progress/:userId/:subjectId", getSubjectProgress)
    
    // AI-powered features
    router.POST("/api/ai/explain", explainConcept)
    router.POST("/api/ai/generate-flashcards", generateFlashcardsFromText)
    router.POST("/api/ai/study-plan", generateStudyPlan)
    
    port := getEnv("API_PORT", getEnv("PORT", ""))
    log.Printf("Starting Study Buddy API on port %s", port)
    router.Run(":" + port)
}

func createSubject(c *gin.Context) {
    var subject Subject
    if err := c.BindJSON(&subject); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    subject.ID = uuid.New().String()
    subject.CreatedAt = time.Now()
    
    query := `INSERT INTO subjects (id, user_id, name, description, color, icon, created_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
    _, err := db.Exec(query, subject.ID, subject.UserID, subject.Name, 
                      subject.Description, subject.Color, subject.Icon, subject.CreatedAt)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, subject)
}

func getUserSubjects(c *gin.Context) {
    userId := c.Param("userId")
    
    query := `SELECT id, user_id, name, description, color, icon, created_at 
              FROM subjects WHERE user_id = $1 ORDER BY created_at DESC`
    rows, err := db.Query(query, userId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var subjects []Subject
    for rows.Next() {
        var subject Subject
        err := rows.Scan(&subject.ID, &subject.UserID, &subject.Name, 
                        &subject.Description, &subject.Color, &subject.Icon, &subject.CreatedAt)
        if err != nil {
            continue
        }
        subjects = append(subjects, subject)
    }
    
    c.JSON(200, subjects)
}

func updateSubject(c *gin.Context) {
    id := c.Param("id")
    var subject Subject
    if err := c.BindJSON(&subject); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    query := `UPDATE subjects SET name = $1, description = $2, color = $3, icon = $4, 
              updated_at = CURRENT_TIMESTAMP WHERE id = $5`
    _, err := db.Exec(query, subject.Name, subject.Description, subject.Color, subject.Icon, id)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Subject updated successfully"})
}

func deleteSubject(c *gin.Context) {
    id := c.Param("id")
    
    query := `DELETE FROM subjects WHERE id = $1`
    _, err := db.Exec(query, id)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Subject deleted successfully"})
}

func createStudyMaterial(c *gin.Context) {
    var material StudyMaterial
    if err := c.BindJSON(&material); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    material.ID = uuid.New().String()
    material.CreatedAt = time.Now()
    
    tagsJSON, _ := json.Marshal(material.Tags)
    
    query := `INSERT INTO study_materials (id, user_id, subject_id, title, content, material_type, tags, created_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := db.Exec(query, material.ID, material.UserID, material.SubjectID, 
                      material.Title, material.Content, material.MaterialType, tagsJSON, material.CreatedAt)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Trigger N8N workflow to generate embeddings
    go triggerEmbeddingGeneration(material.ID, material.Content)
    
    c.JSON(201, material)
}

func getUserMaterials(c *gin.Context) {
    userId := c.Param("userId")
    
    query := `SELECT id, user_id, subject_id, title, content, material_type, tags, created_at 
              FROM study_materials WHERE user_id = $1 ORDER BY created_at DESC`
    rows, err := db.Query(query, userId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var materials []StudyMaterial
    for rows.Next() {
        var material StudyMaterial
        var tagsJSON []byte
        err := rows.Scan(&material.ID, &material.UserID, &material.SubjectID, 
                        &material.Title, &material.Content, &material.MaterialType, 
                        &tagsJSON, &material.CreatedAt)
        if err != nil {
            continue
        }
        json.Unmarshal(tagsJSON, &material.Tags)
        materials = append(materials, material)
    }
    
    c.JSON(200, materials)
}

func getSubjectMaterials(c *gin.Context) {
    userId := c.Param("userId")
    subjectId := c.Param("subjectId")
    
    query := `SELECT id, user_id, subject_id, title, content, material_type, tags, created_at 
              FROM study_materials WHERE user_id = $1 AND subject_id = $2 ORDER BY created_at DESC`
    rows, err := db.Query(query, userId, subjectId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var materials []StudyMaterial
    for rows.Next() {
        var material StudyMaterial
        var tagsJSON []byte
        err := rows.Scan(&material.ID, &material.UserID, &material.SubjectID, 
                        &material.Title, &material.Content, &material.MaterialType, 
                        &tagsJSON, &material.CreatedAt)
        if err != nil {
            continue
        }
        json.Unmarshal(tagsJSON, &material.Tags)
        materials = append(materials, material)
    }
    
    c.JSON(200, materials)
}

func updateStudyMaterial(c *gin.Context) {
    id := c.Param("id")
    var material StudyMaterial
    if err := c.BindJSON(&material); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    tagsJSON, _ := json.Marshal(material.Tags)
    
    query := `UPDATE study_materials SET title = $1, content = $2, material_type = $3, tags = $4, 
              updated_at = CURRENT_TIMESTAMP WHERE id = $5`
    _, err := db.Exec(query, material.Title, material.Content, material.MaterialType, tagsJSON, id)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Material updated successfully"})
}

func deleteStudyMaterial(c *gin.Context) {
    id := c.Param("id")
    
    query := `DELETE FROM study_materials WHERE id = $1`
    _, err := db.Exec(query, id)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Material deleted successfully"})
}

func createFlashcard(c *gin.Context) {
    var flashcard Flashcard
    if err := c.BindJSON(&flashcard); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    flashcard.ID = uuid.New().String()
    flashcard.CreatedAt = time.Now()
    
    tagsJSON, _ := json.Marshal(flashcard.Tags)
    
    query := `INSERT INTO flashcards (id, user_id, subject_id, front, back, difficulty_level, tags, created_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := db.Exec(query, flashcard.ID, flashcard.UserID, flashcard.SubjectID, 
                      flashcard.Front, flashcard.Back, flashcard.Difficulty, tagsJSON, flashcard.CreatedAt)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Initialize spaced repetition for this flashcard
    initQuery := `INSERT INTO spaced_repetition (user_id, flashcard_id) VALUES ($1, $2)`
    db.Exec(initQuery, flashcard.UserID, flashcard.ID)
    
    c.JSON(201, flashcard)
}

func getUserFlashcards(c *gin.Context) {
    userId := c.Param("userId")
    
    query := `SELECT id, user_id, subject_id, front, back, difficulty_level, tags, created_at 
              FROM flashcards WHERE user_id = $1 ORDER BY created_at DESC`
    rows, err := db.Query(query, userId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var flashcards []Flashcard
    for rows.Next() {
        var flashcard Flashcard
        var tagsJSON []byte
        err := rows.Scan(&flashcard.ID, &flashcard.UserID, &flashcard.SubjectID, 
                        &flashcard.Front, &flashcard.Back, &flashcard.Difficulty, 
                        &tagsJSON, &flashcard.CreatedAt)
        if err != nil {
            continue
        }
        json.Unmarshal(tagsJSON, &flashcard.Tags)
        flashcards = append(flashcards, flashcard)
    }
    
    c.JSON(200, flashcards)
}

func getSubjectFlashcards(c *gin.Context) {
    userId := c.Param("userId")
    subjectId := c.Param("subjectId")
    
    query := `SELECT id, user_id, subject_id, front, back, difficulty_level, tags, created_at 
              FROM flashcards WHERE user_id = $1 AND subject_id = $2 ORDER BY created_at DESC`
    rows, err := db.Query(query, userId, subjectId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var flashcards []Flashcard
    for rows.Next() {
        var flashcard Flashcard
        var tagsJSON []byte
        err := rows.Scan(&flashcard.ID, &flashcard.UserID, &flashcard.SubjectID, 
                        &flashcard.Front, &flashcard.Back, &flashcard.Difficulty, 
                        &tagsJSON, &flashcard.CreatedAt)
        if err != nil {
            continue
        }
        json.Unmarshal(tagsJSON, &flashcard.Tags)
        flashcards = append(flashcards, flashcard)
    }
    
    c.JSON(200, flashcards)
}

func getDueFlashcards(c *gin.Context) {
    userId := c.Param("userId")
    
    query := `SELECT f.id, f.user_id, f.subject_id, f.front, f.back, f.difficulty_level, f.tags, f.created_at 
              FROM flashcards f 
              JOIN spaced_repetition sr ON f.id = sr.flashcard_id 
              WHERE f.user_id = $1 AND sr.next_review_date <= CURRENT_DATE 
              ORDER BY sr.next_review_date ASC`
    rows, err := db.Query(query, userId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var flashcards []Flashcard
    for rows.Next() {
        var flashcard Flashcard
        var tagsJSON []byte
        err := rows.Scan(&flashcard.ID, &flashcard.UserID, &flashcard.SubjectID, 
                        &flashcard.Front, &flashcard.Back, &flashcard.Difficulty, 
                        &tagsJSON, &flashcard.CreatedAt)
        if err != nil {
            continue
        }
        json.Unmarshal(tagsJSON, &flashcard.Tags)
        flashcards = append(flashcards, flashcard)
    }
    
    c.JSON(200, flashcards)
}

func reviewFlashcard(c *gin.Context) {
    flashcardId := c.Param("id")
    
    var review struct {
        UserID  string `json:"user_id"`
        Quality int    `json:"quality"`
    }
    
    if err := c.BindJSON(&review); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Update spaced repetition data based on SM-2 algorithm
    // This is a simplified version - the actual implementation would be more complex
    query := `UPDATE spaced_repetition 
              SET repetitions = repetitions + 1,
                  last_reviewed_at = CURRENT_TIMESTAMP,
                  quality_rating = $1,
                  next_review_date = CURRENT_DATE + interval_days * INTERVAL '1 day',
                  interval_days = CASE 
                      WHEN $1 >= 3 THEN interval_days * 2
                      ELSE 1
                  END
              WHERE flashcard_id = $2 AND user_id = $3`
    
    _, err := db.Exec(query, review.Quality, flashcardId, review.UserID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Review recorded successfully"})
}

func startStudySession(c *gin.Context) {
    var session StudySession
    if err := c.BindJSON(&session); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    session.ID = uuid.New().String()
    session.StartedAt = time.Now()
    
    query := `INSERT INTO study_sessions (id, user_id, subject_id, session_type, started_at) 
              VALUES ($1, $2, $3, $4, $5)`
    _, err := db.Exec(query, session.ID, session.UserID, session.SubjectID, 
                      session.SessionType, session.StartedAt)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, session)
}

func endStudySession(c *gin.Context) {
    sessionId := c.Param("id")
    
    var update struct {
        CardsReviewed     int `json:"cards_reviewed"`
        QuestionsAnswered int `json:"questions_answered"`
        CorrectAnswers    int `json:"correct_answers"`
        Notes             string `json:"notes"`
    }
    
    if err := c.BindJSON(&update); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    query := `UPDATE study_sessions 
              SET ended_at = CURRENT_TIMESTAMP,
                  duration_minutes = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - started_at))/60,
                  cards_reviewed = $1,
                  questions_answered = $2,
                  correct_answers = $3,
                  notes = $4
              WHERE id = $5`
    
    _, err := db.Exec(query, update.CardsReviewed, update.QuestionsAnswered, 
                      update.CorrectAnswers, update.Notes, sessionId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Session ended successfully"})
}

func getUserSessions(c *gin.Context) {
    userId := c.Param("userId")
    
    query := `SELECT id, user_id, subject_id, session_type, started_at, ended_at, 
              cards_reviewed, questions_answered, correct_answers 
              FROM study_sessions WHERE user_id = $1 ORDER BY started_at DESC`
    rows, err := db.Query(query, userId)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    var sessions []StudySession
    for rows.Next() {
        var session StudySession
        err := rows.Scan(&session.ID, &session.UserID, &session.SubjectID, 
                        &session.SessionType, &session.StartedAt, &session.EndedAt,
                        &session.CardsReviewed, &session.QuestionsAnswered, &session.CorrectAnswers)
        if err != nil {
            continue
        }
        sessions = append(sessions, session)
    }
    
    c.JSON(200, sessions)
}

func getStudyStats(c *gin.Context) {
    userId := c.Param("userId")
    
    // Get various statistics
    stats := make(map[string]interface{})
    
    // Total study time
    var totalMinutes sql.NullInt64
    db.QueryRow(`SELECT SUM(duration_minutes) FROM study_sessions WHERE user_id = $1`, userId).Scan(&totalMinutes)
    stats["total_study_minutes"] = totalMinutes.Int64
    
    // Total flashcards
    var totalCards int
    db.QueryRow(`SELECT COUNT(*) FROM flashcards WHERE user_id = $1`, userId).Scan(&totalCards)
    stats["total_flashcards"] = totalCards
    
    // Cards due for review
    var dueCards int
    db.QueryRow(`SELECT COUNT(*) FROM spaced_repetition WHERE user_id = $1 AND next_review_date <= CURRENT_DATE`, userId).Scan(&dueCards)
    stats["cards_due_for_review"] = dueCards
    
    // Current streak
    var streak int
    db.QueryRow(`SELECT MAX(streak_days) FROM learning_progress WHERE user_id = $1`, userId).Scan(&streak)
    stats["current_streak"] = streak
    
    c.JSON(200, stats)
}

func generateQuiz(c *gin.Context) {
    var request struct {
        UserID    string `json:"user_id"`
        SubjectID string `json:"subject_id"`
        Count     int    `json:"count"`
    }
    
    if err := c.BindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // This would trigger the N8N workflow for quiz generation
    // For now, return a simple response
    c.JSON(200, gin.H{
        "message": "Quiz generation initiated",
        "workflow_triggered": true,
    })
}

func submitQuizAnswers(c *gin.Context) {
    var submission struct {
        UserID    string                   `json:"user_id"`
        SessionID string                   `json:"session_id"`
        Answers   []map[string]interface{} `json:"answers"`
    }
    
    if err := c.BindJSON(&submission); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Process quiz answers and calculate score
    // This would be more complex in production
    c.JSON(200, gin.H{
        "score": 85,
        "correct": 17,
        "total": 20,
    })
}

func getQuizHistory(c *gin.Context) {
    userId := c.Param("userId")
    
    // Return quiz history
    c.JSON(200, gin.H{
        "user_id": userId,
        "quizzes": []interface{}{},
    })
}

func getLearningAnalytics(c *gin.Context) {
    userId := c.Param("userId")
    
    // Return comprehensive learning analytics
    c.JSON(200, gin.H{
        "user_id": userId,
        "analytics": map[string]interface{}{
            "study_patterns": []interface{}{},
            "performance_trends": []interface{}{},
            "subject_mastery": []interface{}{},
        },
    })
}

func getSubjectProgress(c *gin.Context) {
    userId := c.Param("userId")
    subjectId := c.Param("subjectId")
    
    // Return subject-specific progress
    c.JSON(200, gin.H{
        "user_id": userId,
        "subject_id": subjectId,
        "progress": map[string]interface{}{
            "completion_percentage": 65,
            "cards_mastered": 42,
            "average_score": 87.5,
        },
    })
}

func explainConcept(c *gin.Context) {
    var request struct {
        Concept string `json:"concept"`
        Context string `json:"context"`
    }
    
    if err := c.BindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // This would trigger an AI workflow
    c.JSON(200, gin.H{
        "explanation": "AI-generated explanation would appear here",
        "examples": []string{},
        "related_concepts": []string{},
    })
}

func generateFlashcardsFromText(c *gin.Context) {
    var request struct {
        UserID    string `json:"user_id"`
        SubjectID string `json:"subject_id"`
        Text      string `json:"text"`
    }
    
    if err := c.BindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // This would trigger the flashcard generation workflow
    c.JSON(200, gin.H{
        "message": "Flashcard generation initiated",
        "estimated_cards": 10,
    })
}

func generateStudyPlan(c *gin.Context) {
    var request struct {
        UserID   string   `json:"user_id"`
        Goals    []string `json:"goals"`
        Deadline string   `json:"deadline"`
    }
    
    if err := c.BindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // This would trigger the study plan generation workflow
    c.JSON(200, gin.H{
        "message": "Study plan generation initiated",
        "plan_id": uuid.New().String(),
    })
}

func triggerEmbeddingGeneration(materialId, content string) {
    // This would call the N8N webhook to generate embeddings
    // Implementation would depend on N8N setup
    log.Printf("Triggering embedding generation for material %s", materialId)
}