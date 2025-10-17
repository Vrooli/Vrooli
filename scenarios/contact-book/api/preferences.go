package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CommunicationPreference represents learned preferences for a contact
type CommunicationPreference struct {
	PersonID                  string                 `json:"person_id"`
	PreferredChannels         []string               `json:"preferred_channels"`         // email, phone, text, social
	BestTimeToContact         *TimeWindow            `json:"best_time_to_contact"`       // Preferred contact hours
	ResponseTimePatterns      map[string]float64     `json:"response_time_patterns"`     // Channel -> avg hours
	CommunicationFrequency    string                 `json:"communication_frequency"`    // daily, weekly, monthly, rare
	TopicAffinities           map[string]float64     `json:"topic_affinities"`          // Topic -> interest score
	ConversationStyle         string                 `json:"conversation_style"`         // formal, casual, brief, detailed
	PreferredMeetingDuration  int                    `json:"preferred_meeting_duration"` // Minutes
	LanguagePreferences       []string               `json:"language_preferences"`
	TimezoneSensitivity       bool                   `json:"timezone_sensitivity"`       // Respects timezone boundaries
	LastAnalyzed              time.Time              `json:"last_analyzed"`
	ConfidenceScore           float64                `json:"confidence_score"`           // 0-1, how confident in preferences
	DataPoints                int                    `json:"data_points"`                // Number of interactions analyzed
	Metadata                  map[string]interface{} `json:"metadata"`
}

// TimeWindow represents a preferred time range
type TimeWindow struct {
	StartHour int    `json:"start_hour"` // 0-23
	EndHour   int    `json:"end_hour"`   // 0-23
	Timezone  string `json:"timezone"`
	DaysOfWeek []string `json:"days_of_week"` // mon, tue, wed, thu, fri, sat, sun
}

// CommunicationInteraction represents a single communication event for learning
type CommunicationInteraction struct {
	PersonID      string    `json:"person_id"`
	Channel       string    `json:"channel"`       // email, phone, text, meeting
	Direction     string    `json:"direction"`     // inbound, outbound
	Timestamp     time.Time `json:"timestamp"`
	ResponseTime  *float64  `json:"response_time"` // Hours to response
	Duration      *int      `json:"duration"`      // Minutes for calls/meetings
	Topics        []string  `json:"topics"`        // Keywords/topics discussed
	Tone          *string   `json:"tone"`          // formal, casual, friendly, business
	Sentiment     *float64  `json:"sentiment"`     // -1 to 1
	Engagement    *float64  `json:"engagement"`    // 0 to 1
	WasSuccessful bool      `json:"was_successful"` // Did it achieve its goal
}

// LearnPreferences analyzes communication history to learn preferences
func LearnPreferences(personID string) (*CommunicationPreference, error) {
	// Get communication history for the person
	query := `
		SELECT
			channel, direction, communication_date, response_latency_hours,
			tone, subject_category, metadata
		FROM communication_history
		WHERE to_person_id = $1
		AND communication_date > NOW() - INTERVAL '6 months'
		ORDER BY communication_date DESC
		LIMIT 100`

	rows, err := db.Query(query, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []CommunicationInteraction
	channelCounts := make(map[string]int)
	responseTimes := make(map[string][]float64)
	hourCounts := make(map[int]int)
	topicCounts := make(map[string]int)
	dayOfWeekCounts := make(map[string]int)
	toneMap := make(map[string]int)

	for rows.Next() {
		var i CommunicationInteraction
		var metadata interface{}
		var responseHours sql.NullInt64
		var tone, subjectCategory sql.NullString

		err := rows.Scan(
			&i.Channel, &i.Direction, &i.Timestamp,
			&responseHours, &tone, &subjectCategory, &metadata,
		)
		if err != nil {
			log.Printf("Error scanning communication history: %v", err)
			continue
		}

		i.PersonID = personID

		// Convert nullable fields
		if responseHours.Valid {
			hours := float64(responseHours.Int64)
			i.ResponseTime = &hours
			responseTimes[i.Channel] = append(responseTimes[i.Channel], hours)
		}

		if tone.Valid {
			toneStr := tone.String
			i.Tone = &toneStr
			toneMap[toneStr]++
		}

		// Use subject_category as topic
		if subjectCategory.Valid {
			i.Topics = []string{subjectCategory.String}
			topicCounts[subjectCategory.String]++
		}

		// Count patterns
		channelCounts[i.Channel]++
		hour := i.Timestamp.Hour()
		hourCounts[hour]++
		dayOfWeek := i.Timestamp.Weekday().String()[:3]
		dayOfWeekCounts[strings.ToLower(dayOfWeek)]++

		// Count topic mentions
		for _, topic := range i.Topics {
			topicCounts[topic]++
		}

		interactions = append(interactions, i)
	}

	// Analyze patterns
	pref := &CommunicationPreference{
		PersonID:             personID,
		LastAnalyzed:         time.Now(),
		DataPoints:           len(interactions),
		ResponseTimePatterns: make(map[string]float64),
		TopicAffinities:      make(map[string]float64),
		Metadata:             make(map[string]interface{}),
	}

	// Determine preferred channels (top 2)
	pref.PreferredChannels = getTopChannels(channelCounts, 2)

	// Calculate average response times per channel
	for channel, times := range responseTimes {
		if len(times) > 0 {
			sum := 0.0
			for _, t := range times {
				sum += t
			}
			pref.ResponseTimePatterns[channel] = sum / float64(len(times))
		}
	}

	// Determine best contact time window
	pref.BestTimeToContact = calculateBestTimeWindow(hourCounts, dayOfWeekCounts)

	// Calculate communication frequency
	pref.CommunicationFrequency = calculateFrequency(interactions)

	// Calculate topic affinities
	totalTopics := 0
	for _, count := range topicCounts {
		totalTopics += count
	}
	if totalTopics > 0 {
		for topic, count := range topicCounts {
			pref.TopicAffinities[topic] = float64(count) / float64(totalTopics)
		}
	}

	// Determine conversation style based on tone patterns
	pref.ConversationStyle = determineMostCommonTone(toneMap)

	// Calculate preferred meeting duration (default to 30 min if no data)
	pref.PreferredMeetingDuration = 30

	// Calculate confidence score based on data points and recency
	pref.ConfidenceScore = calculateConfidence(interactions)

	// Save learned preferences
	if err := savePreferences(pref); err != nil {
		log.Printf("Failed to save preferences: %v", err)
	}

	return pref, nil
}

// Helper functions for preference learning

func getTopChannels(counts map[string]int, n int) []string {
	type channelCount struct {
		channel string
		count   int
	}

	var channels []channelCount
	for ch, count := range counts {
		channels = append(channels, channelCount{ch, count})
	}

	// Sort by count (simple bubble sort for small data)
	for i := 0; i < len(channels); i++ {
		for j := i + 1; j < len(channels); j++ {
			if channels[j].count > channels[i].count {
				channels[i], channels[j] = channels[j], channels[i]
			}
		}
	}

	result := []string{}
	for i := 0; i < n && i < len(channels); i++ {
		result = append(result, channels[i].channel)
	}
	return result
}

func calculateBestTimeWindow(hourCounts map[int]int, dayOfWeekCounts map[string]int) *TimeWindow {
	if len(hourCounts) == 0 {
		return nil
	}

	// Find peak hours (simple approach: hours with >average activity)
	totalCount := 0
	for _, count := range hourCounts {
		totalCount += count
	}
	avgCount := float64(totalCount) / 24.0

	startHour := -1
	endHour := -1
	for h := 0; h < 24; h++ {
		if float64(hourCounts[h]) > avgCount {
			if startHour == -1 {
				startHour = h
			}
			endHour = h
		}
	}

	if startHour == -1 {
		// Default business hours
		startHour = 9
		endHour = 17
	}

	// Get top days of week
	var topDays []string
	for day, count := range dayOfWeekCounts {
		if count > 0 {
			topDays = append(topDays, day)
		}
	}

	return &TimeWindow{
		StartHour:  startHour,
		EndHour:    endHour,
		Timezone:   "UTC", // Could be enhanced to detect timezone
		DaysOfWeek: topDays,
	}
}

func calculateFrequency(interactions []CommunicationInteraction) string {
	if len(interactions) < 2 {
		return "rare"
	}

	// Calculate average days between interactions
	totalDays := 0.0
	for i := 1; i < len(interactions); i++ {
		days := interactions[i-1].Timestamp.Sub(interactions[i].Timestamp).Hours() / 24
		totalDays += days
	}
	avgDays := totalDays / float64(len(interactions)-1)

	switch {
	case avgDays < 2:
		return "daily"
	case avgDays < 8:
		return "weekly"
	case avgDays < 35:
		return "monthly"
	default:
		return "rare"
	}
}

func determineConversationStyle(interactions []CommunicationInteraction) string {
	var totalEngagement float64
	var engagementCount int
	var totalDuration int
	var durationCount int

	for _, i := range interactions {
		if i.Engagement != nil {
			totalEngagement += *i.Engagement
			engagementCount++
		}
		if i.Duration != nil {
			totalDuration += *i.Duration
			durationCount++
		}
	}

	avgEngagement := 0.5
	if engagementCount > 0 {
		avgEngagement = totalEngagement / float64(engagementCount)
	}

	avgDuration := 30
	if durationCount > 0 {
		avgDuration = totalDuration / durationCount
	}

	// Determine style based on engagement and duration
	if avgEngagement > 0.7 && avgDuration > 30 {
		return "detailed"
	} else if avgEngagement > 0.7 && avgDuration <= 30 {
		return "casual"
	} else if avgEngagement <= 0.7 && avgDuration <= 15 {
		return "brief"
	} else {
		return "formal"
	}
}

func determineMostCommonTone(toneMap map[string]int) string {
	maxCount := 0
	mostCommon := "casual"

	for tone, count := range toneMap {
		if count > maxCount {
			maxCount = count
			mostCommon = tone
		}
	}

	return mostCommon
}

func calculateConfidence(interactions []CommunicationInteraction) float64 {
	if len(interactions) == 0 {
		return 0.0
	}

	// Base confidence on number of data points
	dataPointScore := math.Min(float64(len(interactions))/50.0, 1.0)

	// Factor in recency (interactions in last 30 days)
	recentCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	for _, i := range interactions {
		if i.Timestamp.After(thirtyDaysAgo) {
			recentCount++
		}
	}
	recencyScore := math.Min(float64(recentCount)/10.0, 1.0)

	// Combine scores
	return (dataPointScore*0.7 + recencyScore*0.3)
}

func savePreferences(pref *CommunicationPreference) error {
	// Store in person's computed_signals JSONB field
	prefsJSON, err := json.Marshal(pref)
	if err != nil {
		return err
	}

	query := `
		UPDATE persons
		SET
			computed_signals = jsonb_set(
				COALESCE(computed_signals, '{}'::jsonb),
				'{communication_preferences}',
				$1::jsonb
			),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	_, err = db.Exec(query, string(prefsJSON), pref.PersonID)
	return err
}

// API endpoint to get learned preferences
func getCommunicationPreferences(c *gin.Context) {
	personID := c.Param("id")

	// Try to get from cache first
	var computedSignals json.RawMessage
	err := db.QueryRow(`
		SELECT computed_signals
		FROM persons
		WHERE id = $1`,
		personID).Scan(&computedSignals)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences"})
		return
	}

	// Check if preferences exist in computed signals
	var signals map[string]interface{}
	if computedSignals != nil {
		json.Unmarshal(computedSignals, &signals)
		if prefs, ok := signals["communication_preferences"]; ok {
			c.JSON(http.StatusOK, prefs)
			return
		}
	}

	// If not cached or outdated, learn preferences
	prefs, err := LearnPreferences(personID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to learn preferences"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// API endpoint to manually record a communication interaction
func recordCommunication(c *gin.Context) {
	var interaction CommunicationInteraction
	if err := c.ShouldBindJSON(&interaction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert into communication_history table
	query := `
		INSERT INTO communication_history (
			to_person_id, channel, direction, communication_date,
			response_latency_hours, tone, subject_category, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	var responseHours *int
	if interaction.ResponseTime != nil {
		hours := int(*interaction.ResponseTime)
		responseHours = &hours
	}

	// Determine tone based on interaction metadata
	tone := "casual"
	if interaction.Tone != nil {
		tone = *interaction.Tone
	}

	// Extract primary topic as subject category
	subjectCategory := "general"
	if len(interaction.Topics) > 0 {
		subjectCategory = interaction.Topics[0]
	}

	_, err := db.Exec(query,
		interaction.PersonID, interaction.Channel, interaction.Direction,
		interaction.Timestamp, responseHours, tone, subjectCategory,
		"{}", // Empty metadata for now
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record communication"})
		return
	}

	// Trigger preference learning in background
	go func() {
		if _, err := LearnPreferences(interaction.PersonID); err != nil {
			log.Printf("Failed to update preferences for %s: %v", interaction.PersonID, err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{"message": "Communication recorded successfully"})
}