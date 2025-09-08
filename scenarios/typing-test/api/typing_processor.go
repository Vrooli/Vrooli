package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TypingProcessor struct {
	db *sql.DB
}

type SessionStats struct {
	SessionID       string  `json:"sessionId"`
	WPM             int     `json:"wpm"`
	Accuracy        float64 `json:"accuracy"`
	CharactersTyped int     `json:"charactersTyped"`
	ErrorCount      int     `json:"errorCount"`
	TimeSpent       int     `json:"timeSpent"`
	Difficulty      string  `json:"difficulty"`
	TextCompleted   bool    `json:"textCompleted"`
}

type PerformanceAnalysis struct {
	PerformanceLevel string   `json:"performanceLevel"`
	ImprovementScore int      `json:"improvementScore"`
	ProblemAreas     []string `json:"problemAreas"`
	Recommendations  []string `json:"recommendations"`
}

type ProcessedStats struct {
	SessionID        string              `json:"sessionId"`
	Timestamp        string              `json:"timestamp"`
	Metrics          StatsMetrics        `json:"metrics"`
	Analysis         PerformanceAnalysis `json:"analysis"`
	PersonalizedTips []Tip               `json:"personalizedTips"`
	NextGoals        Goals               `json:"nextGoals"`
}

type StatsMetrics struct {
	WPM             int     `json:"wpm"`
	Accuracy        float64 `json:"accuracy"`
	ErrorRate       float64 `json:"errorRate"`
	CharactersTyped int     `json:"charactersTyped"`
	ErrorCount      int     `json:"errorCount"`
	TimeSpent       int     `json:"timeSpent"`
}

type Tip struct {
	Category string `json:"category"`
	Tip      string `json:"tip"`
	Priority string `json:"priority"`
}

type Goals struct {
	WPM      int `json:"wpm"`
	Accuracy int `json:"accuracy"`
}

type CoachingResponse struct {
	Feedback        string   `json:"feedback"`
	Tips            []string `json:"tips"`
	EncouragingNote string   `json:"encouragingNote"`
	NextChallenge   string   `json:"nextChallenge"`
}

type LeaderboardEntry struct {
	Rank      int       `json:"rank"`
	Name      string    `json:"name"`
	Score     int       `json:"score"`
	WPM       int       `json:"wpm"`
	Accuracy  float64   `json:"accuracy"`
	Date      time.Time `json:"date"`
	IsCurrentUser bool   `json:"isCurrentUser,omitempty"`
}

func NewTypingProcessor(db *sql.DB) *TypingProcessor {
	return &TypingProcessor{
		db: db,
	}
}

// ProcessStats analyzes typing statistics and provides feedback (replaces n8n typing-stats-processor workflow)
func (tp *TypingProcessor) ProcessStats(ctx context.Context, stats SessionStats) (ProcessedStats, error) {
	// Calculate performance metrics
	errorRate := 100.0 - stats.Accuracy
	
	// Analyze performance
	analysis := tp.analyzePerformance(stats)
	
	// Generate personalized tips
	tips := tp.generateTips(stats, analysis)
	
	// Calculate next goals
	nextGoals := Goals{
		WPM:      int(math.Min(float64(stats.WPM+10), 100)),
		Accuracy: int(math.Min(stats.Accuracy+5, 100)),
	}
	
	// Save session to database
	sessionData, _ := json.Marshal(map[string]interface{}{
		"metrics":  stats,
		"analysis": analysis,
	})
	
	query := `
		INSERT INTO game_sessions (session_id, session_data, started_at, duration_seconds)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id) DO UPDATE SET
			session_data = $2,
			duration_seconds = $4`
	
	_, err := tp.db.ExecContext(ctx, query, 
		stats.SessionID, string(sessionData), time.Now(), stats.TimeSpent)
	
	if err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to save session: %v\n", err)
	}
	
	return ProcessedStats{
		SessionID: stats.SessionID,
		Timestamp: time.Now().Format(time.RFC3339),
		Metrics: StatsMetrics{
			WPM:             stats.WPM,
			Accuracy:        stats.Accuracy,
			ErrorRate:       errorRate,
			CharactersTyped: stats.CharactersTyped,
			ErrorCount:      stats.ErrorCount,
			TimeSpent:       stats.TimeSpent,
		},
		Analysis:         analysis,
		PersonalizedTips: tips[:min(3, len(tips))], // Top 3 tips
		NextGoals:        nextGoals,
	}, nil
}

// ProvideCoaching generates personalized coaching feedback (replaces n8n typing-coach workflow)
func (tp *TypingProcessor) ProvideCoaching(ctx context.Context, stats SessionStats, difficulty string) CoachingResponse {
	response := CoachingResponse{
		Tips: []string{},
	}
	
	// Analyze performance level
	if stats.WPM >= 80 && stats.Accuracy >= 95 {
		response.Feedback = "Outstanding performance! You're typing at an expert level."
		response.EncouragingNote = "You're among the top typists! Keep pushing your limits."
		response.NextChallenge = "Try typing complex technical documents or code to further challenge yourself."
		response.Tips = []string{
			"Practice with specialized vocabulary in your field",
			"Try typing in different languages",
			"Challenge yourself with speed typing competitions",
		}
	} else if stats.WPM >= 60 && stats.Accuracy >= 90 {
		response.Feedback = "Great job! You're typing at an advanced level."
		response.EncouragingNote = "You're making excellent progress. Keep up the momentum!"
		response.NextChallenge = "Focus on maintaining accuracy while gradually increasing your speed."
		response.Tips = []string{
			"Practice with longer, more complex sentences",
			"Work on special characters and punctuation",
			"Try typing without looking at the screen occasionally",
		}
	} else if stats.WPM >= 40 && stats.Accuracy >= 85 {
		response.Feedback = "Good progress! You're at an intermediate level."
		response.EncouragingNote = "You're improving steadily. Every session makes you better!"
		response.NextChallenge = "Work on building muscle memory for common word patterns."
		response.Tips = []string{
			"Practice common word combinations",
			"Focus on problem keys that slow you down",
			"Maintain a steady rhythm while typing",
		}
	} else {
		response.Feedback = "You're building a solid foundation. Keep practicing!"
		response.EncouragingNote = "Everyone starts somewhere. Your dedication will pay off!"
		response.NextChallenge = "Focus on accuracy first, speed will naturally follow."
		response.Tips = []string{
			"Use proper finger positioning on the home row",
			"Type slowly but accurately to build muscle memory",
			"Practice for 10-15 minutes daily for best results",
		}
	}
	
	// Add specific feedback based on metrics
	if stats.Accuracy < 80 {
		response.Tips = append(response.Tips, "Slow down slightly to improve accuracy - speed will come with practice")
	}
	if stats.WPM < 30 {
		response.Tips = append(response.Tips, "Focus on not looking at the keyboard while typing")
	}
	
	return response
}

// GenerateAdaptiveText creates personalized practice text (replaces n8n adaptive-text-generator workflow)
func (tp *TypingProcessor) GenerateAdaptiveText(ctx context.Context, req AdaptiveTextRequest) AdaptiveTextResponse {
	var text string
	wordCount := 50 // default
	
	// Adjust word count based on text length preference
	switch req.TextLength {
	case "short":
		wordCount = 25
	case "medium":
		wordCount = 50
	case "long":
		wordCount = 100
	}
	
	// Generate text based on difficulty and user level
	if req.Difficulty == "easy" || req.UserLevel == "beginner" {
		text = tp.generateEasyText(wordCount, req.TargetWords)
	} else if req.Difficulty == "medium" || req.UserLevel == "intermediate" {
		text = tp.generateMediumText(wordCount, req.TargetWords, req.ProblemChars)
	} else {
		text = tp.generateHardText(wordCount, req.ProblemChars)
	}
	
	// If there are previous mistakes, incorporate those words
	if len(req.PreviousMistakes) > 0 {
		// Add problematic words to the text
		for _, mistake := range req.PreviousMistakes {
			if mistake.ErrorCount > 2 {
				text = strings.Replace(text, " ", " "+mistake.Word+" ", 1)
			}
		}
	}
	
	return AdaptiveTextResponse{
		Text:       text,
		WordCount:  len(strings.Fields(text)),
		Difficulty: req.Difficulty,
		IsAdaptive: true,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

// ManageLeaderboard handles leaderboard operations (replaces n8n leaderboard-manager workflow)
func (tp *TypingProcessor) ManageLeaderboard(ctx context.Context, period string, userID string) ([]LeaderboardEntry, error) {
	var timeFilter string
	switch period {
	case "daily":
		timeFilter = "WHERE created_at > NOW() - INTERVAL '1 day'"
	case "weekly":
		timeFilter = "WHERE created_at > NOW() - INTERVAL '7 days'"
	case "monthly":
		timeFilter = "WHERE created_at > NOW() - INTERVAL '30 days'"
	default:
		timeFilter = "" // all time
	}
	
	query := fmt.Sprintf(`
		SELECT 
			ROW_NUMBER() OVER (ORDER BY score DESC) as rank,
			name, score, wpm, accuracy, created_at, user_id
		FROM scores
		%s
		ORDER BY score DESC
		LIMIT 100`, timeFilter)
	
	rows, err := tp.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leaderboard: %w", err)
	}
	defer rows.Close()
	
	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		var dbUserID sql.NullString
		
		err := rows.Scan(&entry.Rank, &entry.Name, &entry.Score, 
			&entry.WPM, &entry.Accuracy, &entry.Date, &dbUserID)
		if err != nil {
			continue
		}
		
		// Mark current user's entry
		if dbUserID.Valid && dbUserID.String == userID {
			entry.IsCurrentUser = true
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// AddScore adds a new score to the leaderboard
func (tp *TypingProcessor) AddScore(ctx context.Context, score Score) error {
	query := `
		INSERT INTO scores (name, score, wpm, accuracy, max_combo, difficulty, mode, created_at, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	userID := uuid.New().String() // Generate if not provided
	
	_, err := tp.db.ExecContext(ctx, query,
		score.Name, score.Score, score.WPM, score.Accuracy,
		score.MaxCombo, score.Difficulty, score.Mode, time.Now(), userID)
	
	return err
}

// Helper methods

func (tp *TypingProcessor) analyzePerformance(stats SessionStats) PerformanceAnalysis {
	var performanceLevel string
	var recommendations []string
	problemAreas := []string{}
	
	// Determine performance level
	if stats.WPM >= 80 && stats.Accuracy >= 95 {
		performanceLevel = "expert"
		recommendations = append(recommendations, 
			"Challenge yourself with harder texts",
			"Try speed typing competitions")
	} else if stats.WPM >= 60 && stats.Accuracy >= 90 {
		performanceLevel = "advanced"
		recommendations = append(recommendations,
			"Focus on maintaining accuracy at higher speeds",
			"Practice with technical or complex texts")
	} else if stats.WPM >= 40 && stats.Accuracy >= 85 {
		performanceLevel = "intermediate"
		recommendations = append(recommendations,
			"Work on increasing speed without sacrificing accuracy",
			"Practice with diverse text types")
	} else {
		performanceLevel = "beginner"
		recommendations = append(recommendations,
			"Focus on accuracy before speed",
			"Practice with easier texts first",
			"Use proper finger positioning on home row")
	}
	
	// Identify problem areas
	if stats.Accuracy < 85 {
		problemAreas = append(problemAreas, "accuracy")
	}
	if stats.WPM < 30 {
		problemAreas = append(problemAreas, "speed")
	}
	if stats.ErrorCount > 20 {
		problemAreas = append(problemAreas, "consistency")
	}
	
	// Calculate improvement score
	improvementScore := int(math.Round(
		(float64(stats.WPM)*0.6 + stats.Accuracy*0.4) * 
		map[bool]float64{true: 1.0, false: 0.8}[stats.TextCompleted]))
	
	return PerformanceAnalysis{
		PerformanceLevel: performanceLevel,
		ImprovementScore: improvementScore,
		ProblemAreas:     problemAreas,
		Recommendations:  recommendations,
	}
}

func (tp *TypingProcessor) generateTips(stats SessionStats, analysis PerformanceAnalysis) []Tip {
	tips := []Tip{}
	
	// Speed-based tips
	if stats.WPM < 30 {
		tips = append(tips, Tip{
			Category: "speed",
			Tip:      "Try typing without looking at the keyboard",
			Priority: "high",
		})
		tips = append(tips, Tip{
			Category: "speed",
			Tip:      "Practice common word combinations daily",
			Priority: "medium",
		})
	} else if stats.WPM < 50 {
		tips = append(tips, Tip{
			Category: "speed",
			Tip:      "Focus on rhythm and flow rather than individual keys",
			Priority: "medium",
		})
	}
	
	// Accuracy-based tips
	if stats.Accuracy < 80 {
		tips = append(tips, Tip{
			Category: "accuracy",
			Tip:      "Slow down and focus on hitting the right keys",
			Priority: "high",
		})
		tips = append(tips, Tip{
			Category: "accuracy",
			Tip:      "Practice problem letters separately",
			Priority: "high",
		})
	} else if stats.Accuracy < 90 {
		tips = append(tips, Tip{
			Category: "accuracy",
			Tip:      "Review your common mistakes and practice those patterns",
			Priority: "medium",
		})
	}
	
	// General improvement tips
	if analysis.PerformanceLevel == "expert" {
		tips = append(tips, Tip{
			Category: "challenge",
			Tip:      "Try typing with punctuation and special characters",
			Priority: "low",
		})
		tips = append(tips, Tip{
			Category: "challenge",
			Tip:      "Practice typing code or technical documentation",
			Priority: "low",
		})
	} else {
		tips = append(tips, Tip{
			Category: "practice",
			Tip:      "Aim for 15-20 minutes of practice daily",
			Priority: "medium",
		})
	}
	
	return tips
}

func (tp *TypingProcessor) generateEasyText(wordCount int, targetWords []string) string {
	commonWords := []string{
		"the", "and", "for", "are", "but", "not", "you", "can", "her", "was",
		"one", "our", "out", "day", "get", "has", "him", "his", "how", "its",
		"may", "new", "now", "old", "see", "two", "way", "who", "boy", "did",
		"car", "let", "put", "say", "she", "too", "use", "her", "make", "good",
		"time", "know", "year", "work", "back", "call", "came", "high", "need", "feel",
	}
	
	// Include target words if provided
	if len(targetWords) > 0 {
		commonWords = append(commonWords, targetWords...)
	}
	
	// Generate text
	text := []string{}
	for i := 0; i < wordCount; i++ {
		word := commonWords[rand.Intn(len(commonWords))]
		text = append(text, word)
	}
	
	return strings.Join(text, " ")
}

func (tp *TypingProcessor) generateMediumText(wordCount int, targetWords []string, problemChars []string) string {
	sentences := []string{
		"The quick brown fox jumps over the lazy dog",
		"Practice makes perfect when learning to type",
		"Technology advances rapidly in modern times",
		"Reading books expands your knowledge and vocabulary",
		"Exercise regularly to maintain good health",
		"Learning new skills requires patience and dedication",
		"Travel broadens the mind and creates memories",
		"Music has the power to change our mood instantly",
	}
	
	// Add sentences with problem characters
	if len(problemChars) > 0 {
		for _, char := range problemChars {
			sentences = append(sentences, fmt.Sprintf("Practice typing the letter %s repeatedly", char))
		}
	}
	
	// Build text from sentences
	text := ""
	wordsAdded := 0
	for wordsAdded < wordCount {
		sentence := sentences[rand.Intn(len(sentences))]
		text += sentence + " "
		wordsAdded += len(strings.Fields(sentence))
	}
	
	// Trim to exact word count
	words := strings.Fields(text)
	if len(words) > wordCount {
		words = words[:wordCount]
	}
	
	return strings.Join(words, " ")
}

func (tp *TypingProcessor) generateHardText(wordCount int, problemChars []string) string {
	complexSentences := []string{
		"Pseudopseudohypoparathyroidism is a complex medical condition",
		"The @ symbol, #hashtag, and $price all require shift key usage",
		"JavaScript's async/await syntax simplifies promise handling",
		"Quantum computing revolutionizes cryptographic algorithms",
		"Anthropomorphization unnecessarily complicates explanations",
		"Debugging regex patterns: ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
		"The CEO's Q3 P&L showed 47.3% YoY growth @ $2.8M EBITDA",
		"PostgreSQL's JSONB type offers GIN indexing for performant queries",
	}
	
	// Build complex text
	text := ""
	wordsAdded := 0
	for wordsAdded < wordCount {
		sentence := complexSentences[rand.Intn(len(complexSentences))]
		text += sentence + " "
		wordsAdded += len(strings.Fields(sentence))
	}
	
	// Trim to word count
	words := strings.Fields(text)
	if len(words) > wordCount {
		words = words[:wordCount]
	}
	
	return strings.Join(words, " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}