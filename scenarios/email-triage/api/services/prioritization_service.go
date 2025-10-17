package services

import (
	"strings"
	"time"
	"regexp"
	
	"email-triage/models"
)

// PrioritizationService handles email priority scoring
type PrioritizationService struct {
	// Weight factors for different signals
	senderWeight    float64
	subjectWeight   float64
	contentWeight   float64
	recipientWeight float64
	timeWeight      float64
}

// NewPrioritizationService creates a new PrioritizationService instance
func NewPrioritizationService() *PrioritizationService {
	return &PrioritizationService{
		senderWeight:    0.3,  // 30% weight for sender importance
		subjectWeight:   0.25, // 25% weight for subject keywords
		contentWeight:   0.2,  // 20% weight for content analysis
		recipientWeight: 0.15, // 15% weight for recipient patterns
		timeWeight:      0.1,  // 10% weight for time sensitivity
	}
}

// CalculatePriority calculates a priority score (0-1) for an email
func (ps *PrioritizationService) CalculatePriority(email *models.ProcessedEmail) float64 {
	var totalScore float64
	
	// Sender importance score
	senderScore := ps.calculateSenderScore(email.SenderEmail)
	totalScore += senderScore * ps.senderWeight
	
	// Subject urgency score
	subjectScore := ps.calculateSubjectScore(email.Subject)
	totalScore += subjectScore * ps.subjectWeight
	
	// Content analysis score
	contentScore := ps.calculateContentScore(email.FullBody)
	totalScore += contentScore * ps.contentWeight
	
	// Recipient pattern score (direct vs CC, number of recipients)
	recipientScore := ps.calculateRecipientScore(email.RecipientEmails)
	totalScore += recipientScore * ps.recipientWeight
	
	// Time sensitivity score (newer emails get slight boost)
	timeScore := ps.calculateTimeScore(email.ProcessedAt)
	totalScore += timeScore * ps.timeWeight
	
	// Ensure score is between 0 and 1
	if totalScore > 1.0 {
		totalScore = 1.0
	}
	if totalScore < 0 {
		totalScore = 0
	}
	
	return totalScore
}

// calculateSenderScore evaluates sender importance
func (ps *PrioritizationService) calculateSenderScore(sender string) float64 {
	senderLower := strings.ToLower(sender)
	
	// VIP senders
	vipDomains := []string{"@ceo.", "@executive.", "@board.", "@president."}
	for _, vip := range vipDomains {
		if strings.Contains(senderLower, vip) {
			return 1.0
		}
	}
	
	// Important internal domains
	if strings.HasSuffix(senderLower, "@example.com") {
		return 0.8
	}
	
	// Known important external domains
	importantDomains := []string{"@client.", "@partner.", "@investor."}
	for _, domain := range importantDomains {
		if strings.Contains(senderLower, domain) {
			return 0.7
		}
	}
	
	// Automated/newsletter senders (lower priority)
	automatedKeywords := []string{"noreply", "no-reply", "newsletter", "notification", "automated", "system"}
	for _, keyword := range automatedKeywords {
		if strings.Contains(senderLower, keyword) {
			return 0.2
		}
	}
	
	// Default score for unknown senders
	return 0.5
}

// calculateSubjectScore evaluates subject line urgency
func (ps *PrioritizationService) calculateSubjectScore(subject string) float64 {
	subjectLower := strings.ToLower(subject)
	
	// Urgent keywords
	urgentKeywords := []string{"urgent", "asap", "immediate", "critical", "emergency", "important"}
	urgentCount := 0
	for _, keyword := range urgentKeywords {
		if strings.Contains(subjectLower, keyword) {
			urgentCount++
		}
	}
	if urgentCount > 0 {
		return 0.9 + (0.1 * float64(urgentCount-1) / float64(len(urgentKeywords)))
	}
	
	// Action-required keywords
	actionKeywords := []string{"action required", "response needed", "please review", "approval needed", "deadline"}
	for _, keyword := range actionKeywords {
		if strings.Contains(subjectLower, keyword) {
			return 0.8
		}
	}
	
	// Meeting/calendar keywords
	meetingKeywords := []string{"meeting", "call", "appointment", "schedule", "interview", "demo"}
	for _, keyword := range meetingKeywords {
		if strings.Contains(subjectLower, keyword) {
			return 0.7
		}
	}
	
	// FYI/Newsletter keywords (lower priority)
	fyiKeywords := []string{"fyi", "newsletter", "digest", "update", "announcement", "blog"}
	for _, keyword := range fyiKeywords {
		if strings.Contains(subjectLower, keyword) {
			return 0.3
		}
	}
	
	// Check for ALL CAPS (often indicates importance or spam)
	if strings.ToUpper(subject) == subject && len(subject) > 5 {
		// Could be important or spam, give moderate score
		return 0.6
	}
	
	return 0.5
}

// calculateContentScore analyzes email body content
func (ps *PrioritizationService) calculateContentScore(content string) float64 {
	if content == "" {
		return 0.5
	}
	
	contentLower := strings.ToLower(content)
	score := 0.5
	
	// Check for deadline mentions
	deadlinePattern := regexp.MustCompile(`\b(deadline|due date|due by|expires?|by \d+[/-]\d+)\b`)
	if deadlinePattern.MatchString(contentLower) {
		score = 0.8
	}
	
	// Check for monetary amounts (often important)
	moneyPattern := regexp.MustCompile(`\$[\d,]+|\d+\s*(dollars?|usd|eur|gbp)`)
	if moneyPattern.MatchString(contentLower) {
		score = maxFloat64(score, 0.7)
	}
	
	// Check for question marks (might need response)
	questionCount := strings.Count(content, "?")
	if questionCount > 0 {
		score = maxFloat64(score, 0.6 + minFloat64(0.3, float64(questionCount)*0.1))
	}
	
	// Check for attachments mentioned
	if strings.Contains(contentLower, "attached") || strings.Contains(contentLower, "attachment") {
		score = maxFloat64(score, 0.65)
	}
	
	// Length analysis (very short or very long might be important)
	wordCount := len(strings.Fields(content))
	if wordCount < 20 {
		// Very short, might be urgent directive
		score = maxFloat64(score, 0.6)
	} else if wordCount > 500 {
		// Long detailed email, might be important documentation
		score = maxFloat64(score, 0.55)
	}
	
	return score
}

// calculateRecipientScore evaluates recipient patterns
func (ps *PrioritizationService) calculateRecipientScore(recipients []string) float64 {
	if len(recipients) == 0 {
		return 0.5
	}
	
	// Direct email to single recipient (more personal, likely more important)
	if len(recipients) == 1 {
		return 0.8
	}
	
	// Small group (2-5 recipients)
	if len(recipients) <= 5 {
		return 0.6
	}
	
	// Large group (likely mass email)
	if len(recipients) > 20 {
		return 0.2
	}
	
	// Medium group
	return 0.4
}

// calculateTimeScore gives newer emails a slight priority boost
func (ps *PrioritizationService) calculateTimeScore(processedAt time.Time) float64 {
	now := time.Now()
	age := now.Sub(processedAt)
	
	// Emails from last hour
	if age < time.Hour {
		return 1.0
	}
	
	// Emails from today
	if age < 24*time.Hour {
		return 0.8
	}
	
	// Emails from this week
	if age < 7*24*time.Hour {
		return 0.6
	}
	
	// Emails from this month
	if age < 30*24*time.Hour {
		return 0.4
	}
	
	// Older emails
	return 0.2
}

// Helper functions
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}