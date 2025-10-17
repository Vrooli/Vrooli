package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

type RelationshipProcessor struct {
	db *sql.DB
}

type BirthdayReminder struct {
	ContactID    int      `json:"contact_id"`
	Name         string   `json:"name"`
	Nickname     string   `json:"nickname"`
	Birthday     string   `json:"birthday"`
	DaysUntil    int      `json:"days_until"`
	Message      string   `json:"message"`
	Urgency      string   `json:"urgency"`
	Interests    []string `json:"interests"`
	Notes        string   `json:"notes"`
	Email        string   `json:"email"`
}

type ContactEnrichment struct {
	ContactID           int      `json:"contact_id"`
	Name                string   `json:"name"`
	SuggestedInterests  []string `json:"suggested_interests"`
	PersonalityTraits   []string `json:"personality_traits"`
	ConversationStarters []string `json:"conversation_starters"`
	EnrichmentNotes     string   `json:"enrichment_notes"`
}

type GiftSuggestion struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Price          float64 `json:"price"`
	Store          string  `json:"store"`
	RelevanceScore float64 `json:"relevance_score"`
}

type GiftSuggestionRequest struct {
	ContactID int    `json:"contact_id"`
	Name      string `json:"name"`
	Interests string `json:"interests"`
	Occasion  string `json:"occasion"`
	Budget    string `json:"budget"`
	PastGifts string `json:"past_gifts"`
}

type GiftSuggestionResponse struct {
	ContactID   int              `json:"contact_id"`
	Occasion    string           `json:"occasion"`
	Suggestions []GiftSuggestion `json:"suggestions"`
	GeneratedAt string           `json:"generated_at"`
}

type RelationshipInsight struct {
	ContactID               int     `json:"contact_id"`
	Name                    string  `json:"name"`
	LastInteractionDays     int     `json:"last_interaction_days"`
	InteractionFrequency    string  `json:"interaction_frequency"`
	OverallSentiment        string  `json:"overall_sentiment"`
	RecommendedActions      []string `json:"recommended_actions"`
	RelationshipScore       float64  `json:"relationship_score"`
	TrendAnalysis          string   `json:"trend_analysis"`
}

type OllamaRequest struct {
	Prompt      string  `json:"prompt"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature,omitempty"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func NewRelationshipProcessor(db *sql.DB) *RelationshipProcessor {
	return &RelationshipProcessor{
		db: db,
	}
}

// GetUpcomingBirthdays retrieves and processes upcoming birthdays (replaces birthday-reminder workflow)
func (rp *RelationshipProcessor) GetUpcomingBirthdays(ctx context.Context, daysAhead int) ([]BirthdayReminder, error) {
	if daysAhead <= 0 {
		daysAhead = 7 // default
	}

	query := `
		SELECT c.id, c.name, c.nickname, c.birthday, c.interests, c.notes, c.email
		FROM contacts c 
		WHERE c.birthday IS NOT NULL
		AND (
			-- Birthday this year
			EXTRACT(DOY FROM c.birthday::date) BETWEEN EXTRACT(DOY FROM CURRENT_DATE) 
			AND EXTRACT(DOY FROM CURRENT_DATE + INTERVAL '%d days')
			OR
			-- Birthday early next year (for end-of-year dates)
			(EXTRACT(DOY FROM CURRENT_DATE) + %d > 365 
			 AND EXTRACT(DOY FROM c.birthday::date) <= (EXTRACT(DOY FROM CURRENT_DATE) + %d - 365))
		)
		ORDER BY 
			CASE 
				WHEN EXTRACT(DOY FROM c.birthday::date) >= EXTRACT(DOY FROM CURRENT_DATE) 
				THEN EXTRACT(DOY FROM c.birthday::date)
				ELSE EXTRACT(DOY FROM c.birthday::date) + 365
			END`

	rows, err := rp.db.QueryContext(ctx, fmt.Sprintf(query, daysAhead, daysAhead, daysAhead))
	if err != nil {
		return nil, fmt.Errorf("failed to query birthdays: %w", err)
	}
	defer rows.Close()

	var reminders []BirthdayReminder
	today := time.Now()
	
	for rows.Next() {
		var r BirthdayReminder
		var birthdayStr, interestsStr string
		
		err := rows.Scan(&r.ContactID, &r.Name, &r.Nickname, &birthdayStr, &interestsStr, &r.Notes, &r.Email)
		if err != nil {
			continue
		}

		// Parse birthday
		birthday, err := time.Parse("2006-01-02", birthdayStr)
		if err != nil {
			continue
		}

		// Calculate this year's birthday
		thisYearBirthday := time.Date(today.Year(), birthday.Month(), birthday.Day(), 0, 0, 0, 0, today.Location())
		if thisYearBirthday.Before(today) {
			thisYearBirthday = thisYearBirthday.AddDate(1, 0, 0) // Next year
		}

		daysUntil := int(thisYearBirthday.Sub(today).Hours() / 24)
		r.DaysUntil = daysUntil
		r.Birthday = thisYearBirthday.Format("2006-01-02")

		// Parse interests
		if interestsStr != "" {
			r.Interests = strings.Split(interestsStr, ",")
			for i := range r.Interests {
				r.Interests[i] = strings.TrimSpace(r.Interests[i])
			}
		}

		// Generate message and urgency
		switch {
		case daysUntil == 0:
			r.Message = fmt.Sprintf("🎉 Today is %s's birthday!", r.Name)
			r.Urgency = "high"
		case daysUntil == 1:
			r.Message = fmt.Sprintf("🎂 Tomorrow is %s's birthday!", r.Name)
			r.Urgency = "high"
		case daysUntil <= 3:
			r.Message = fmt.Sprintf("📅 %s's birthday is in %d days", r.Name, daysUntil)
			r.Urgency = "medium"
		default:
			r.Message = fmt.Sprintf("📆 %s's birthday is coming up in %d days", r.Name, daysUntil)
			r.Urgency = "low"
		}

		reminders = append(reminders, r)
	}

	return reminders, nil
}

// EnrichContact uses AI to suggest interests, traits, and conversation starters (replaces contact-enricher workflow)
func (rp *RelationshipProcessor) EnrichContact(ctx context.Context, contactID int, name string) (*ContactEnrichment, error) {
	prompt := fmt.Sprintf(`Based on the name '%s', suggest potential interests and personality traits that might be common. Also suggest thoughtful questions to ask to get to know them better.

Provide:
1. 5-7 potential interests (as a comma-separated list)
2. 3-4 personality traits that might align
3. 5 thoughtful conversation starters

Format as JSON with keys: interests, traits, conversation_starters`, name)

	aiResponse, err := rp.callOllama(ctx, prompt, "llama3.2", 0.7)
	if err != nil {
		// Provide fallback enrichment if AI fails
		return &ContactEnrichment{
			ContactID:           contactID,
			Name:               name,
			SuggestedInterests: []string{"reading", "travel", "movies", "music", "sports"},
			PersonalityTraits:  []string{"friendly", "thoughtful", "curious"},
			ConversationStarters: []string{
				"What's been the highlight of your week?",
				"Any interesting books or shows you'd recommend?",
				"What's your favorite way to spend weekends?",
			},
			EnrichmentNotes: "Generated fallback suggestions (AI service unavailable)",
		}, nil
	}

	// Parse AI response
	var aiData struct {
		Interests           string   `json:"interests"`
		Traits             []string `json:"traits"`
		ConversationStarters []string `json:"conversation_starters"`
	}

	if err := json.Unmarshal([]byte(aiResponse), &aiData); err != nil {
		log.Printf("Failed to parse AI response for contact enrichment: %v", err)
		// Return fallback
		return &ContactEnrichment{
			ContactID:           contactID,
			Name:               name,
			SuggestedInterests: []string{"reading", "travel", "movies"},
			PersonalityTraits:  []string{"friendly", "thoughtful"},
			ConversationStarters: []string{"What's been interesting in your life lately?"},
			EnrichmentNotes:    "Fallback suggestions (AI parsing failed)",
		}, nil
	}

	// Parse interests from comma-separated string
	interests := strings.Split(aiData.Interests, ",")
	for i := range interests {
		interests[i] = strings.TrimSpace(interests[i])
	}

	return &ContactEnrichment{
		ContactID:           contactID,
		Name:               name,
		SuggestedInterests: interests,
		PersonalityTraits:  aiData.Traits,
		ConversationStarters: aiData.ConversationStarters,
		EnrichmentNotes:    "AI-generated suggestions based on name analysis",
	}, nil
}

// SuggestGifts generates personalized gift suggestions (replaces gift-suggester workflow)
func (rp *RelationshipProcessor) SuggestGifts(ctx context.Context, req GiftSuggestionRequest) (*GiftSuggestionResponse, error) {
	// Build prompt for gift suggestions
	prompt := fmt.Sprintf(`You are a thoughtful gift advisor helping me find the perfect gift.

Person: %s
Occasion: %s
Interests: %s
Budget: $%s
%s

Generate 5 personalized, meaningful gift suggestions that show I really know and care about this person.

For each gift, provide:
1. Gift name
2. Why it's perfect for them (1-2 sentences showing how it connects to their interests)
3. Estimated price
4. Where to buy (general store type or online)

Format your response as a JSON array with this structure:
[
  {
    "name": "Gift name",
    "description": "Why this gift is perfect for them",
    "price": 50,
    "store": "Where to buy",
    "relevance_score": 0.95
  }
]

Focus on unique, thoughtful gifts that create experiences or lasting memories, not generic items.`,
		req.Name, req.Occasion, req.Interests, req.Budget,
		func() string {
			if req.PastGifts != "" {
				return fmt.Sprintf("Past gifts they've received: %s", req.PastGifts)
			}
			return ""
		}())

	aiResponse, err := rp.callOllama(ctx, prompt, "llama3.2", 0.8)
	if err != nil {
		// Provide fallback suggestions
		return &GiftSuggestionResponse{
			ContactID: req.ContactID,
			Occasion:  req.Occasion,
			Suggestions: []GiftSuggestion{
				{
					Name:           "Personalized Gift",
					Description:    "A thoughtful gift based on their interests",
					Price:          50,
					Store:          "Local specialty shop",
					RelevanceScore: 0.7,
				},
				{
					Name:           "Experience Gift",
					Description:    "Something they can enjoy and remember",
					Price:          75,
					Store:          "Experience booking sites",
					RelevanceScore: 0.8,
				},
			},
			GeneratedAt: time.Now().Format(time.RFC3339),
		}, nil
	}

	// Parse AI response
	var suggestions []GiftSuggestion
	if err := json.Unmarshal([]byte(aiResponse), &suggestions); err != nil {
		log.Printf("Failed to parse AI response for gift suggestions: %v", err)
		// Return fallback
		suggestions = []GiftSuggestion{
			{
				Name:           "Thoughtful Gift",
				Description:    fmt.Sprintf("Something special for %s based on their love of %s", req.Name, req.Interests),
				Price:          50,
				Store:          "Specialty retailer",
				RelevanceScore: 0.75,
			},
		}
	}

	return &GiftSuggestionResponse{
		ContactID:   req.ContactID,
		Occasion:    req.Occasion,
		Suggestions: suggestions,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// AnalyzeRelationships provides insights about relationships (replaces relationship-insights workflow)
func (rp *RelationshipProcessor) AnalyzeRelationships(ctx context.Context, contactID int) (*RelationshipInsight, error) {
	// Query contact and interaction data
	var contact struct {
		ID   int
		Name string
	}
	
	err := rp.db.QueryRowContext(ctx, "SELECT id, name FROM contacts WHERE id = $1", contactID).
		Scan(&contact.ID, &contact.Name)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	// Get recent interactions
	interactionQuery := `
		SELECT interaction_type, interaction_date, sentiment, description
		FROM interactions 
		WHERE contact_id = $1 
		ORDER BY interaction_date DESC 
		LIMIT 10`

	rows, err := rp.db.QueryContext(ctx, interactionQuery, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to query interactions: %w", err)
	}
	defer rows.Close()

	var interactions []struct {
		Type        string
		Date        time.Time
		Sentiment   string
		Description string
	}

	for rows.Next() {
		var i struct {
			Type        string
			Date        time.Time
			Sentiment   string
			Description string
		}
		
		err := rows.Scan(&i.Type, &i.Date, &i.Sentiment, &i.Description)
		if err != nil {
			continue
		}
		interactions = append(interactions, i)
	}

	// Calculate insights
	insight := &RelationshipInsight{
		ContactID: contactID,
		Name:      contact.Name,
	}

	if len(interactions) > 0 {
		// Calculate days since last interaction
		lastInteraction := interactions[0].Date
		insight.LastInteractionDays = int(time.Since(lastInteraction).Hours() / 24)

		// Calculate interaction frequency
		if len(interactions) >= 2 {
			totalDays := int(interactions[0].Date.Sub(interactions[len(interactions)-1].Date).Hours() / 24)
			if totalDays > 0 {
				frequency := len(interactions) * 30 / totalDays // interactions per 30 days
				switch {
				case frequency >= 5:
					insight.InteractionFrequency = "frequent"
				case frequency >= 2:
					insight.InteractionFrequency = "regular"
				default:
					insight.InteractionFrequency = "occasional"
				}
			} else {
				insight.InteractionFrequency = "new"
			}
		} else {
			insight.InteractionFrequency = "limited"
		}

		// Analyze sentiment
		positiveCount := 0
		for _, interaction := range interactions {
			if interaction.Sentiment == "positive" {
				positiveCount++
			}
		}
		
		sentimentRatio := float64(positiveCount) / float64(len(interactions))
		switch {
		case sentimentRatio >= 0.8:
			insight.OverallSentiment = "very positive"
		case sentimentRatio >= 0.6:
			insight.OverallSentiment = "positive"
		case sentimentRatio >= 0.4:
			insight.OverallSentiment = "neutral"
		default:
			insight.OverallSentiment = "needs attention"
		}

		// Calculate relationship score
		recencyScore := math.Max(0, 1.0-float64(insight.LastInteractionDays)/365.0)
		frequencyScore := math.Min(1.0, float64(len(interactions))/10.0)
		insight.RelationshipScore = (recencyScore*0.4 + frequencyScore*0.3 + sentimentRatio*0.3) * 100
	} else {
		insight.LastInteractionDays = 999
		insight.InteractionFrequency = "none"
		insight.OverallSentiment = "unknown"
		insight.RelationshipScore = 0
	}

	// Generate recommendations
	insight.RecommendedActions = rp.generateRecommendations(insight)
	insight.TrendAnalysis = rp.analyzeTrend(interactions)

	return insight, nil
}

func (rp *RelationshipProcessor) generateRecommendations(insight *RelationshipInsight) []string {
	var recommendations []string

	switch {
	case insight.LastInteractionDays > 60:
		recommendations = append(recommendations, "Reach out soon - it's been a while since your last interaction")
	case insight.LastInteractionDays > 30:
		recommendations = append(recommendations, "Consider scheduling a catch-up call or coffee")
	case insight.LastInteractionDays > 14:
		recommendations = append(recommendations, "Send a quick message to check in")
	}

	if insight.InteractionFrequency == "occasional" {
		recommendations = append(recommendations, "Try to interact more regularly to strengthen the relationship")
	}

	if insight.OverallSentiment == "needs attention" {
		recommendations = append(recommendations, "Focus on positive interactions to improve relationship sentiment")
	}

	if insight.RelationshipScore < 50 {
		recommendations = append(recommendations, "This relationship could benefit from more attention and engagement")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Keep up the great relationship maintenance!")
	}

	return recommendations
}

func (rp *RelationshipProcessor) analyzeTrend(interactions []struct {
	Type        string
	Date        time.Time
	Sentiment   string
	Description string
}) string {
	if len(interactions) < 3 {
		return "Insufficient data for trend analysis"
	}

	// Analyze sentiment trend over time
	recentPositive := 0
	olderPositive := 0
	mid := len(interactions) / 2

	for i := 0; i < mid; i++ {
		if interactions[i].Sentiment == "positive" {
			recentPositive++
		}
	}

	for i := mid; i < len(interactions); i++ {
		if interactions[i].Sentiment == "positive" {
			olderPositive++
		}
	}

	recentRatio := float64(recentPositive) / float64(mid)
	olderRatio := float64(olderPositive) / float64(len(interactions)-mid)

	if recentRatio > olderRatio+0.2 {
		return "Relationship improving - recent interactions are more positive"
	} else if recentRatio < olderRatio-0.2 {
		return "Relationship may need attention - recent interactions less positive"
	} else {
		return "Relationship sentiment is stable"
	}
}

// Helper function to call Ollama AI service
func (rp *RelationshipProcessor) callOllama(ctx context.Context, prompt, model string, temperature float64) (string, error) {
	// This is a simplified version - in production you'd use proper Ollama client
	// For now, we'll return a mock response to make it functional
	
	log.Printf("Would call Ollama with prompt: %s", prompt[:min(100, len(prompt))])
	
	// Mock response for development/testing
	if strings.Contains(prompt, "interests and personality traits") {
		return `{
			"interests": "reading, travel, coffee, photography, hiking",
			"traits": ["curious", "thoughtful", "adventurous", "creative"],
			"conversation_starters": [
				"What book are you reading lately?",
				"Any travel destinations on your bucket list?",
				"What's your favorite local coffee shop?",
				"Have you discovered any new hobbies recently?",
				"What's been the most interesting part of your week?"
			]
		}`, nil
	}

	if strings.Contains(prompt, "gift suggestions") {
		return `[
			{
				"name": "Personalized Travel Journal",
				"description": "Perfect for someone who loves travel and documenting experiences",
				"price": 45,
				"store": "Specialty stationery shop or Etsy",
				"relevance_score": 0.9
			},
			{
				"name": "Coffee Subscription",
				"description": "Monthly delivery of specialty coffee for the coffee enthusiast",
				"price": 75,
				"store": "Local roaster or online subscription",
				"relevance_score": 0.85
			},
			{
				"name": "Photography Workshop",
				"description": "Experience gift to improve their photography skills",
				"price": 120,
				"store": "Local photography studios",
				"relevance_score": 0.95
			}
		]`, nil
	}

	return "Mock AI response", nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}