package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type QuizProcessor struct {
	db          *pgxpool.Pool
	redis       *redis.Client
	ollamaURL   string
	qdrantURL   string
}

type GeneratedQuestion struct {
	Type          string   `json:"type"`
	Question      string   `json:"question"`
	Options       []string `json:"options,omitempty"`
	CorrectAnswer string   `json:"correct_answer"`
	Explanation   string   `json:"explanation"`
	Difficulty    string   `json:"difficulty"`
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format,omitempty"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func NewQuizProcessor(db *pgxpool.Pool, redis *redis.Client, ollamaURL, qdrantURL string) *QuizProcessor {
	return &QuizProcessor{
		db:        db,
		redis:     redis,
		ollamaURL: ollamaURL,
		qdrantURL: qdrantURL,
	}
}

// GenerateQuizFromContent generates a quiz from provided content using AI (replaces n8n quiz-generator-ai workflow)
func (qp *QuizProcessor) GenerateQuizFromContent(ctx context.Context, req QuizGenerateRequest) (*Quiz, error) {
	// Set defaults
	if req.QuestionCount == 0 {
		req.QuestionCount = 5
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}
	if len(req.QuestionTypes) == 0 {
		req.QuestionTypes = []string{"mcq", "true_false", "short_answer"}
	}

	// Generate questions using Ollama
	questions, err := qp.generateQuestionsWithOllama(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate questions: %w", err)
	}

	// Create quiz structure
	quiz := &Quiz{
		ID:           uuid.New().String(),
		Title:        "Generated Quiz",
		Description:  "AI-generated quiz from provided content",
		Questions:    qp.structureQuestions(questions),
		TimeLimit:    req.QuestionCount * 120, // 2 minutes per question average
		PassingScore: 70,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save quiz to database
	if err := qp.saveQuizToDB(ctx, quiz); err != nil {
		return nil, fmt.Errorf("failed to save quiz: %w", err)
	}

	// Cache quiz in Redis for quick access
	if err := qp.cacheQuiz(ctx, quiz); err != nil {
		// Log error but don't fail the request
		logger.Warnf("Failed to cache quiz: %v", err)
	}

	// Store in vector database for semantic search
	go qp.storeInVectorDB(context.Background(), quiz)

	return quiz, nil
}

func (qp *QuizProcessor) generateQuestionsWithOllama(ctx context.Context, req QuizGenerateRequest) ([]GeneratedQuestion, error) {
	// For now, use fallback questions to test the system
	// TODO: Enable Ollama integration when ready
	return qp.generateFallbackQuestions(req), nil
	
	prompt := fmt.Sprintf(`You are an expert quiz generator. Generate exactly %d questions from the following content.

Content: %s

Requirements:
1. Difficulty level: %s
2. Question types to include: %s
3. Each question must be directly based on the provided content
4. Provide clear, unambiguous questions
5. For multiple choice questions, provide 4 options with only 1 correct answer
6. Include explanations for correct answers

Return the response as a valid JSON array with this structure:
[
  {
    "type": "mcq",
    "question": "question text",
    "options": ["A) option1", "B) option2", "C) option3", "D) option4"],
    "correct_answer": "A",
    "explanation": "explanation text",
    "difficulty": "%s"
  },
  {
    "type": "true_false",
    "question": "question text",
    "correct_answer": "true",
    "explanation": "explanation text",
    "difficulty": "%s"
  },
  {
    "type": "short_answer",
    "question": "question text",
    "correct_answer": "expected answer",
    "explanation": "explanation text",
    "difficulty": "%s"
  }
]`,
		req.QuestionCount,
		req.Content,
		req.Difficulty,
		strings.Join(req.QuestionTypes, ", "),
		req.Difficulty,
		req.Difficulty,
		req.Difficulty,
	)

	ollamaReq := OllamaRequest{
		Model:  "llama2:7b",
		Prompt: prompt,
		Format: "json",
		Stream: false,
	}

	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		qp.ollamaURL+"/api/generate",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	var questions []GeneratedQuestion
	if err := json.Unmarshal([]byte(ollamaResp.Response), &questions); err != nil {
		// If parsing fails, try to generate fallback questions
		questions = qp.generateFallbackQuestions(req)
	}

	return questions, nil
}

func (qp *QuizProcessor) generateFallbackQuestions(req QuizGenerateRequest) []GeneratedQuestion {
	// Generate simple fallback questions if AI fails
	questions := []GeneratedQuestion{}
	
	for i := 0; i < req.QuestionCount; i++ {
		qType := req.QuestionTypes[i%len(req.QuestionTypes)]
		
		switch qType {
		case "mcq":
			questions = append(questions, GeneratedQuestion{
				Type:          "mcq",
				Question:      fmt.Sprintf("Question %d based on the provided content?", i+1),
				Options:       []string{"A) Option 1", "B) Option 2", "C) Option 3", "D) Option 4"},
				CorrectAnswer: "A",
				Explanation:   "This is a placeholder question. Please review the content carefully.",
				Difficulty:    req.Difficulty,
			})
		case "true_false":
			questions = append(questions, GeneratedQuestion{
				Type:          "true_false",
				Question:      fmt.Sprintf("Statement %d about the content is true?", i+1),
				CorrectAnswer: "true",
				Explanation:   "This is a placeholder question. Please review the content carefully.",
				Difficulty:    req.Difficulty,
			})
		case "short_answer":
			questions = append(questions, GeneratedQuestion{
				Type:          "short_answer",
				Question:      fmt.Sprintf("Briefly describe aspect %d of the content", i+1),
				CorrectAnswer: "Sample answer",
				Explanation:   "This is a placeholder question. Please review the content carefully.",
				Difficulty:    req.Difficulty,
			})
		}
	}
	
	return questions
}

func (qp *QuizProcessor) structureQuestions(generated []GeneratedQuestion) []Question {
	questions := make([]Question, len(generated))
	
	for i, gq := range generated {
		points := 1
		switch gq.Difficulty {
		case "hard":
			points = 3
		case "medium":
			points = 2
		}
		
		questions[i] = Question{
			ID:            uuid.New().String(),
			Type:          gq.Type,
			QuestionText:  gq.Question,
			Options:       gq.Options,
			CorrectAnswer: gq.CorrectAnswer,
			Explanation:   gq.Explanation,
			Difficulty:    gq.Difficulty,
			Points:        points,
			OrderIndex:    i + 1,
		}
	}
	
	return questions
}

func (qp *QuizProcessor) saveQuizToDB(ctx context.Context, quiz *Quiz) error {
	// Save quiz metadata
	query := `
		INSERT INTO quiz_generator.quizzes (id, title, description, time_limit, passing_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := qp.db.Exec(ctx, query,
		quiz.ID, quiz.Title, quiz.Description,
		quiz.TimeLimit, quiz.PassingScore,
		quiz.CreatedAt, quiz.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert quiz: %w", err)
	}

	// Save questions
	for _, q := range quiz.Questions {
		optionsJSON, _ := json.Marshal(q.Options)
		answerJSON, _ := json.Marshal(q.CorrectAnswer)
		
		questionQuery := `
			INSERT INTO quiz_generator.questions (id, quiz_id, type, question_text, options, correct_answer, 
				explanation, difficulty, points, order_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
		
		_, err = qp.db.Exec(ctx, questionQuery,
			q.ID, quiz.ID, q.Type, q.QuestionText,
			optionsJSON, answerJSON,
			q.Explanation, q.Difficulty, q.Points, q.OrderIndex)
		if err != nil {
			return fmt.Errorf("failed to insert question: %w", err)
		}
	}

	return nil
}

func (qp *QuizProcessor) cacheQuiz(ctx context.Context, quiz *Quiz) error {
	quizJSON, err := json.Marshal(quiz)
	if err != nil {
		return fmt.Errorf("failed to marshal quiz: %w", err)
	}

	// Cache for 24 hours
	return qp.redis.Set(ctx, fmt.Sprintf("quiz:%s", quiz.ID), quizJSON, 24*time.Hour).Err()
}

func (qp *QuizProcessor) storeInVectorDB(ctx context.Context, quiz *Quiz) {
	// Generate embedding for semantic search
	content := fmt.Sprintf("%s %s", quiz.Title, quiz.Description)
	for _, q := range quiz.Questions {
		content += fmt.Sprintf(" %s", q.QuestionText)
	}

	// Create vector point
	vectorPoint := map[string]interface{}{
		"id":     quiz.ID,
		"vector": qp.generateEmbedding(content),
		"payload": map[string]interface{}{
			"title":       quiz.Title,
			"description": quiz.Description,
			"difficulty":  qp.getAverageDifficulty(quiz.Questions),
			"created_at":  quiz.CreatedAt,
			"tags":        []string{}, // Can be enhanced to extract tags from content
		},
	}

	// Store in Qdrant
	qp.storeVectorPoint(ctx, "quizzes", vectorPoint)
}

func (qp *QuizProcessor) generateEmbedding(text string) []float32 {
	// This is a placeholder - in production, use a real embedding model
	// For now, generate a random 384-dimensional vector
	embedding := make([]float32, 384)
	for i := range embedding {
		embedding[i] = rand.Float32()
	}
	return embedding
}

func (qp *QuizProcessor) storeVectorPoint(ctx context.Context, collection string, point map[string]interface{}) error {
	jsonData, err := json.Marshal(map[string]interface{}{
		"points": []interface{}{point},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal vector point: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/collections/%s/points", qp.qdrantURL, collection),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("qdrant returned status %d", resp.StatusCode)
	}

	return nil
}

func (qp *QuizProcessor) getAverageDifficulty(questions []Question) string {
	difficultyMap := map[string]int{"easy": 0, "medium": 0, "hard": 0}
	
	for _, q := range questions {
		if _, exists := difficultyMap[q.Difficulty]; exists {
			difficultyMap[q.Difficulty]++
		}
	}

	// Find the most common difficulty
	maxCount := 0
	avgDifficulty := "medium"
	for diff, count := range difficultyMap {
		if count > maxCount {
			maxCount = count
			avgDifficulty = diff
		}
	}

	return avgDifficulty
}

// GradeQuiz grades a submitted quiz
func (qp *QuizProcessor) GradeQuiz(ctx context.Context, quizID string, submission QuizSubmitRequest) (*QuizResult, error) {
	// Get quiz from cache or database
	quiz, err := qp.getQuiz(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quiz: %w", err)
	}

	result := &QuizResult{
		Score:          0,
		CorrectAnswers: make(map[string]interface{}),
		Explanations:   make(map[string]string),
	}

	totalPoints := 0
	earnedPoints := 0

	// Grade each response
	for _, resp := range submission.Responses {
		for _, question := range quiz.Questions {
			if question.ID == resp.QuestionID {
				totalPoints += question.Points
				
				// Check if answer is correct
				if qp.checkAnswer(question, resp.Answer) {
					earnedPoints += question.Points
					result.CorrectAnswers[question.ID] = question.CorrectAnswer
				} else {
					result.CorrectAnswers[question.ID] = question.CorrectAnswer
				}
				
				result.Explanations[question.ID] = question.Explanation
				break
			}
		}
	}

	result.Score = earnedPoints
	result.Percentage = float64(earnedPoints) / float64(totalPoints) * 100
	result.Passed = result.Percentage >= float64(quiz.PassingScore)

	// Store result in database
	go qp.storeQuizResult(context.Background(), quizID, submission, result)

	return result, nil
}

func (qp *QuizProcessor) getQuiz(ctx context.Context, quizID string) (*Quiz, error) {
	// Try cache first
	cached, err := qp.redis.Get(ctx, fmt.Sprintf("quiz:%s", quizID)).Result()
	if err == nil {
		var quiz Quiz
		if err := json.Unmarshal([]byte(cached), &quiz); err == nil {
			return &quiz, nil
		}
	}

	// Load from database
	quiz := &Quiz{ID: quizID}
	
	query := `SELECT title, description, time_limit, passing_score, created_at, updated_at 
			  FROM quizzes WHERE id = $1`
	err = qp.db.QueryRow(ctx, query, quizID).Scan(
		&quiz.Title, &quiz.Description, &quiz.TimeLimit,
		&quiz.PassingScore, &quiz.CreatedAt, &quiz.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get quiz: %w", err)
	}

	// Load questions
	questionQuery := `SELECT id, type, question_text, options, correct_answer, explanation, 
					  difficulty, points, order_index
					  FROM questions WHERE quiz_id = $1 ORDER BY order_index`
	rows, err := qp.db.Query(ctx, questionQuery, quizID)
	if err != nil {
		return nil, fmt.Errorf("failed to get questions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var q Question
		var optionsJSON, answerJSON []byte
		
		err := rows.Scan(&q.ID, &q.Type, &q.QuestionText, &optionsJSON, &answerJSON,
			&q.Explanation, &q.Difficulty, &q.Points, &q.OrderIndex)
		if err != nil {
			continue
		}
		
		json.Unmarshal(optionsJSON, &q.Options)
		json.Unmarshal(answerJSON, &q.CorrectAnswer)
		
		quiz.Questions = append(quiz.Questions, q)
	}

	// Cache for future use
	go qp.cacheQuiz(context.Background(), quiz)

	return quiz, nil
}

func (qp *QuizProcessor) checkAnswer(question Question, answer interface{}) bool {
	// Convert both to strings for comparison
	correctStr := fmt.Sprintf("%v", question.CorrectAnswer)
	answerStr := fmt.Sprintf("%v", answer)
	
	// Case-insensitive comparison for text answers
	return strings.EqualFold(correctStr, answerStr)
}

func (qp *QuizProcessor) storeQuizResult(ctx context.Context, quizID string, 
	submission QuizSubmitRequest, result *QuizResult) {
	
	resultID := uuid.New().String()
	responsesJSON, _ := json.Marshal(submission.Responses)
	
	query := `INSERT INTO quiz_results (id, quiz_id, score, percentage, passed, 
			  responses, time_taken, submitted_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	qp.db.Exec(ctx, query, resultID, quizID, result.Score, result.Percentage,
		result.Passed, responsesJSON, submission.TimeTaken, time.Now())
}