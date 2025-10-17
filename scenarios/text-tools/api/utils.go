package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// extractText extracts text from various input types
func extractText(input interface{}) string {
	switch v := input.(type) {
	case string:
		return v
	case map[string]interface{}:
		if url, ok := v["url"].(string); ok {
			// TODO: Implement URL fetching
			return fmt.Sprintf("Content from URL: %s", url)
		}
		if docID, ok := v["document_id"].(string); ok {
			// TODO: Implement document fetching from database
			return fmt.Sprintf("Content from document: %s", docID)
		}
	}
	return ""
}

// Diff operations
func performLineDiff(text1, text2 string, options DiffOptions) ([]Change, float64) {
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")
	
	changes := []Change{}
	similarity := calculateSimilarity(text1, text2)
	
	// Simple line-by-line comparison (stub implementation)
	maxLines := max(len(lines1), len(lines2))
	for i := 0; i < maxLines; i++ {
		line1 := ""
		line2 := ""
		
		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}
		
		if options.IgnoreCase {
			line1 = strings.ToLower(line1)
			line2 = strings.ToLower(line2)
		}
		
		if options.IgnoreWhitespace {
			line1 = strings.TrimSpace(line1)
			line2 = strings.TrimSpace(line2)
		}
		
		if line1 != line2 {
			changeType := "modify"
			if line1 == "" {
				changeType = "add"
			} else if line2 == "" {
				changeType = "remove"
			}
			
			changes = append(changes, Change{
				Type:      changeType,
				LineStart: i + 1,
				LineEnd:   i + 1,
				Content:   line2,
			})
		}
	}
	
	return changes, similarity
}

func performWordDiff(text1, text2 string, options DiffOptions) ([]Change, float64) {
	// Simplified word-level diff
	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)
	
	changes := []Change{}
	similarity := calculateSimilarity(text1, text2)
	
	// Simple word comparison
	if len(words1) != len(words2) {
		changes = append(changes, Change{
			Type:      "modify",
			LineStart: 1,
			LineEnd:   1,
			Content:   fmt.Sprintf("Word count changed from %d to %d", len(words1), len(words2)),
		})
	}
	
	return changes, similarity
}

func performCharDiff(text1, text2 string, options DiffOptions) ([]Change, float64) {
	changes := []Change{}
	similarity := calculateSimilarity(text1, text2)
	
	if text1 != text2 {
		changes = append(changes, Change{
			Type:      "modify",
			LineStart: 1,
			LineEnd:   1,
			Content:   "Character differences detected",
		})
	}
	
	return changes, similarity
}

func performSemanticDiff(text1, text2 string, options DiffOptions, ollamaURL string) ([]Change, float64) {
	// TODO: Implement actual semantic diff using Ollama
	changes := []Change{
		{
			Type:      "semantic",
			LineStart: 1,
			LineEnd:   1,
			Content:   "Semantic analysis not yet implemented",
		},
	}
	similarity := calculateSimilarity(text1, text2)
	return changes, similarity
}

// Search operations
func performSearch(text, pattern string, options SearchOptions) []Match {
	matches := []Match{}
	
	var re *regexp.Regexp
	var err error
	
	if options.Regex {
		flags := ""
		if !options.CaseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + pattern)
		if err != nil {
			return matches
		}
	}
	
	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		var found [][]int
		
		if options.Regex && re != nil {
			found = re.FindAllStringIndex(line, -1)
		} else {
			searchText := line
			searchPattern := pattern
			
			if !options.CaseSensitive {
				searchText = strings.ToLower(line)
				searchPattern = strings.ToLower(pattern)
			}
			
			index := strings.Index(searchText, searchPattern)
			if index != -1 {
				found = [][]int{{index, index + len(pattern)}}
			}
		}
		
		for _, match := range found {
			matches = append(matches, Match{
				Line:    lineNum + 1,
				Column:  match[0] + 1,
				Length:  match[1] - match[0],
				Context: line,
				Score:   1.0,
			})
		}
	}
	
	return matches
}

func performSemanticSearch(text interface{}, pattern string, config *Config) []SemanticMatch {
	// TODO: Implement semantic search using Ollama/Qdrant
	return []SemanticMatch{
		{
			Text:       "Semantic search not yet implemented",
			Score:      0.5,
			Similarity: 0.5,
		},
	}
}

// Transform operations
func applyTransformation(text string, transform Transformation) string {
	switch transform.Type {
	case "case":
		// Handle case transformation with parameters
		if to, ok := transform.Parameters["to"].(string); ok {
			switch to {
			case "upper":
				return strings.ToUpper(text)
			case "lower":
				return strings.ToLower(text)
			case "title":
				return strings.Title(text)
			}
		}
		return text
	case "upper":
		return strings.ToUpper(text)
	case "lower":
		return strings.ToLower(text)
	case "title":
		return strings.Title(text)
	case "encode":
		// Handle encoding transformations
		if encoding, ok := transform.Parameters["type"].(string); ok {
			switch encoding {
			case "base64":
				return base64.StdEncoding.EncodeToString([]byte(text))
			case "url":
				return url.QueryEscape(text)
			}
		}
		return text
	case "format":
		// Handle format transformations
		if format, ok := transform.Parameters["type"].(string); ok {
			switch format {
			case "json":
				// Pretty print as JSON string
				quoted, _ := json.Marshal(text)
				return string(quoted)
			}
		}
		return text
	case "sanitize":
		// Basic HTML tag removal
		re := regexp.MustCompile(`<[^>]*>`)
		return re.ReplaceAllString(text, "")
	default:
		return text
	}
}

func trackTransformationSteps(text string, transforms []Transformation) []string {
	results := []string{text}
	current := text
	
	for _, transform := range transforms {
		current = applyTransformation(current, transform)
		results = append(results, current)
	}
	
	return results
}

// Extract operations
func extractContent(source interface{}, options ExtractOptions) (string, map[string]interface{}) {
	metadata := make(map[string]interface{})
	
	switch v := source.(type) {
	case string:
		metadata["type"] = "string"
		metadata["length"] = len(v)
		return v, metadata
	case map[string]interface{}:
		// Check for direct text content
		if text, ok := v["text"].(string); ok {
			metadata["type"] = "text"
			metadata["length"] = len(text)
			metadata["format"] = "plain"
			return text, metadata
		}
		if url, ok := v["url"].(string); ok {
			// TODO: Implement URL content extraction
			metadata["type"] = "url"
			metadata["source"] = url
			return fmt.Sprintf("Content extracted from URL: %s", url), metadata
		}
		if fileData, ok := v["file"].(string); ok {
			// TODO: Implement file content extraction (base64 decoding)
			metadata["type"] = "file"
			metadata["encoding"] = "base64"
			metadata["file_length"] = len(fileData)
			return fmt.Sprintf("Content extracted from file (length: %d)", len(fileData)), metadata
		}
		if docId, ok := v["document_id"].(string); ok {
			// TODO: Implement document retrieval from database
			metadata["type"] = "document"
			metadata["document_id"] = docId
			return fmt.Sprintf("Content from document: %s", docId), metadata
		}
	}
	
	return "", metadata
}

func extractStructuredData(source interface{}) map[string]interface{} {
	// TODO: Implement structured data extraction
	return map[string]interface{}{
		"status": "not implemented",
	}
}

// Analysis operations
func extractEntities(text string) []Entity {
	// Simple regex-based entity extraction (stub)
	entities := []Entity{}
	
	// Extract email addresses
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailRe.FindAllString(text, -1)
	for _, email := range emails {
		entities = append(entities, Entity{
			Type:       "email",
			Value:      email,
			Confidence: 0.9,
		})
	}
	
	// Extract URLs
	urlRe := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlRe.FindAllString(text, -1)
	for _, url := range urls {
		entities = append(entities, Entity{
			Type:       "url",
			Value:      url,
			Confidence: 0.9,
		})
	}
	
	return entities
}

func analyzeSentiment(text string) Sentiment {
	// Simple sentiment analysis (stub)
	positive := []string{"good", "great", "excellent", "amazing", "wonderful", "fantastic"}
	negative := []string{"bad", "terrible", "awful", "horrible", "disappointing"}
	
	textLower := strings.ToLower(text)
	positiveCount := 0
	negativeCount := 0
	
	for _, word := range positive {
		positiveCount += strings.Count(textLower, word)
	}
	
	for _, word := range negative {
		negativeCount += strings.Count(textLower, word)
	}
	
	score := 0.5 // neutral
	label := "neutral"
	
	if positiveCount > negativeCount {
		score = 0.7
		label = "positive"
	} else if negativeCount > positiveCount {
		score = 0.3
		label = "negative"
	}
	
	return Sentiment{
		Score: score,
		Label: label,
	}
}

func generateSummary(text string, length int) string {
	// Simple summary generation (stub)
	words := strings.Fields(text)
	if length <= 0 || length >= len(words) {
		return text
	}
	
	summary := strings.Join(words[:length], " ")
	return summary + "..."
}

func extractKeywords(text string) []Keyword {
	// Simple keyword extraction based on word frequency
	words := strings.Fields(strings.ToLower(text))
	frequency := make(map[string]int)
	
	// Common stop words to ignore
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}
	
	for _, word := range words {
		cleaned := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(word, "")
		if len(cleaned) > 2 && !stopWords[cleaned] {
			frequency[cleaned]++
		}
	}
	
	keywords := []Keyword{}
	totalWords := float64(len(words))
	
	for word, count := range frequency {
		if count > 1 { // Only include words that appear more than once
			score := float64(count) / totalWords
			keywords = append(keywords, Keyword{
				Word:  word,
				Score: score,
			})
		}
	}
	
	return keywords
}

func detectLanguage(text string) Language {
	// Simple language detection (stub)
	return Language{
		Code:       "en",
		Name:       "English",
		Confidence: 0.8,
	}
}

// calculateTextStatistics calculates basic text statistics
func calculateTextStatistics(text string) TextStatistics {
	stats := TextStatistics{}
	
	// Character count
	stats.CharCount = len(text)
	
	// Line count
	lines := strings.Split(text, "\n")
	stats.LineCount = len(lines)
	
	// Word count and average word length
	words := strings.Fields(text)
	stats.WordCount = len(words)
	
	totalWordLength := 0
	for _, word := range words {
		// Clean punctuation for accurate word length
		cleaned := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(word, "")
		if len(cleaned) > 0 {
			totalWordLength += len(cleaned)
		}
	}
	
	if stats.WordCount > 0 {
		stats.AvgWordLength = float64(totalWordLength) / float64(stats.WordCount)
	}
	
	// Sentence count (simple: count periods, exclamations, and questions)
	sentenceEnders := regexp.MustCompile(`[.!?]+`)
	sentences := sentenceEnders.Split(text, -1)
	// Filter out empty strings
	sentenceCount := 0
	for _, s := range sentences {
		if strings.TrimSpace(s) != "" {
			sentenceCount++
		}
	}
	stats.SentenceCount = sentenceCount
	
	// Paragraph count (separated by double newlines or more)
	paragraphSeparator := regexp.MustCompile(`\n\n+`)
	paragraphs := paragraphSeparator.Split(text, -1)
	paragraphCount := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			paragraphCount++
		}
	}
	stats.ParagraphCount = paragraphCount
	
	// Reading time (average 200 words per minute)
	if stats.WordCount > 0 {
		stats.ReadingTime = (stats.WordCount * 60) / 200 // in seconds
		if stats.ReadingTime < 1 {
			stats.ReadingTime = 1
		}
	}
	
	return stats
}

func generateAIInsights(text string, ollamaURL string) map[string]interface{} {
	// TODO: Implement AI insights using Ollama
	return map[string]interface{}{
		"status":      "not implemented",
		"text_length": len(text),
		"timestamp":   time.Now().Unix(),
	}
}

// Utility functions
func calculateSimilarity(text1, text2 string) float64 {
	if text1 == text2 {
		return 1.0
	}
	
	// Simple similarity based on common character count
	common := 0
	total := max(len(text1), len(text2))
	
	if total == 0 {
		return 1.0
	}
	
	for i := 0; i < min(len(text1), len(text2)); i++ {
		if text1[i] == text2[i] {
			common++
		}
	}
	
	return float64(common) / float64(total)
}

func generateUnifiedPatch(text1, text2 interface{}) string {
	// TODO: Implement unified patch generation
	return "Patch generation not yet implemented"
}

func calculateDiffMetrics(text1, text2 interface{}) map[string]interface{} {
	return map[string]interface{}{
		"lines_added":   0,
		"lines_removed": 0,
		"lines_changed": 0,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}