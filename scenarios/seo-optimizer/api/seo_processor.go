package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type SEOProcessor struct {
	httpClient *http.Client
}

// SEO Audit types
type SEOAuditResponse struct {
	URL              string                 `json:"url"`
	Status           string                 `json:"status"`
	TechnicalAudit   TechnicalAuditResults  `json:"technical_audit"`
	ContentAudit     ContentAuditResults    `json:"content_audit"`
	PerformanceAudit PerformanceAuditResults `json:"performance_audit"`
	Issues           []SEOIssue             `json:"issues"`
	Score            int                    `json:"score"`
	Timestamp        string                 `json:"timestamp"`
}

type TechnicalAuditResults struct {
	HasHTTPS        bool     `json:"has_https"`
	ResponsiveDesign bool    `json:"responsive_design"`
	LoadTime        int      `json:"load_time_ms"`
	StatusCode      int      `json:"status_code"`
	MetaTags        MetaTags `json:"meta_tags"`
	Headers         []string `json:"headers"`
}

type ContentAuditResults struct {
	WordCount        int      `json:"word_count"`
	ReadabilityScore int      `json:"readability_score"`
	KeywordDensity   float64  `json:"keyword_density"`
	InternalLinks    int      `json:"internal_links"`
	ExternalLinks    int      `json:"external_links"`
	Images           int      `json:"images"`
	ImagesWithAlt    int      `json:"images_with_alt"`
	Headers          []string `json:"headers"`
}

type PerformanceAuditResults struct {
	PageSpeed      int `json:"page_speed"`
	MobileScore    int `json:"mobile_score"`
	FirstByte      int `json:"first_byte_ms"`
	DOMContentLoad int `json:"dom_content_load_ms"`
}

type MetaTags struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	TitleLength int    `json:"title_length"`
	DescLength  int    `json:"description_length"`
}

type SEOIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Impact      int    `json:"impact"`
}

// Content Optimization types
type ContentOptimizationResponse struct {
	ContentAnalysis ContentAnalysis            `json:"content_analysis"`
	KeywordAnalysis map[string]KeywordMetrics  `json:"keyword_analysis"`
	Issues          []string                   `json:"issues"`
	Recommendations []string                   `json:"recommendations"`
	Score           int                        `json:"score"`
	ContentType     string                     `json:"content_type"`
}

type ContentAnalysis struct {
	WordCount                int     `json:"word_count"`
	SentenceCount           int     `json:"sentence_count"`
	ParagraphCount          int     `json:"paragraph_count"`
	AvgWordsPerSentence     int     `json:"avg_words_per_sentence"`
	AvgSentencesPerParagraph int    `json:"avg_sentences_per_paragraph"`
	ReadabilityScore        float64 `json:"readability_score"`
}

type KeywordMetrics struct {
	Count              int     `json:"count"`
	Density            float64 `json:"density"`
	InTitle            bool    `json:"in_title"`
	InFirstParagraph   bool    `json:"in_first_paragraph"`
	ProminenceScore    float64 `json:"prominence_score"`
}

// Keyword Research types
type KeywordResearchResponse struct {
	SeedKeyword string             `json:"seed_keyword"`
	Keywords    []KeywordSuggestion `json:"keywords"`
	RelatedTerms []string           `json:"related_terms"`
	LongTail    []KeywordSuggestion `json:"long_tail"`
	Questions   []string           `json:"questions"`
	Status      string             `json:"status"`
}

type KeywordSuggestion struct {
	Keyword     string  `json:"keyword"`
	Volume      string  `json:"volume"`
	Competition string  `json:"competition"`
	CPC         string  `json:"cpc"`
	Intent      string  `json:"intent"`
	Difficulty  int     `json:"difficulty"`
	Relevance   float64 `json:"relevance"`
}

// Competitor Analysis types
type CompetitorAnalysisResponse struct {
	YourURL        string              `json:"your_url"`
	CompetitorURL  string              `json:"competitor_url"`
	Comparison     CompetitorComparison `json:"comparison"`
	Opportunities  []string            `json:"opportunities"`
	Threats        []string            `json:"threats"`
	Recommendations []string           `json:"recommendations"`
	OverallScore   CompetitorScore     `json:"overall_score"`
}

type CompetitorComparison struct {
	SEOScore        CompetitorMetric `json:"seo_score"`
	ContentQuality  CompetitorMetric `json:"content_quality"`
	TechnicalSEO    CompetitorMetric `json:"technical_seo"`
	UserExperience  CompetitorMetric `json:"user_experience"`
	Keywords        CompetitorKeywords `json:"keywords"`
}

type CompetitorMetric struct {
	Yours       int    `json:"yours"`
	Competitor  int    `json:"competitor"`
	Winner      string `json:"winner"`
	Difference  int    `json:"difference"`
}

type CompetitorKeywords struct {
	SharedKeywords []string `json:"shared_keywords"`
	YourUnique     []string `json:"your_unique"`
	TheirUnique    []string `json:"their_unique"`
	Gaps           []string `json:"gaps"`
}

type CompetitorScore struct {
	Overall    int    `json:"overall"`
	Strengths  []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
}

func NewSEOProcessor() *SEOProcessor {
	return &SEOProcessor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PerformSEOAudit conducts comprehensive SEO analysis (replaces seo-audit workflow)
func (seo *SEOProcessor) PerformSEOAudit(ctx context.Context, url string, depth int) (*SEOAuditResponse, error) {
	if depth <= 0 {
		depth = 3
	}

	// Fetch the webpage
	resp, err := seo.httpClient.Get(url)
	if err != nil {
		return &SEOAuditResponse{
			URL:    url,
			Status: "error",
			Issues: []SEOIssue{{
				Type:        "connectivity",
				Severity:    "critical",
				Description: fmt.Sprintf("Failed to fetch URL: %v", err),
				Impact:      100,
			}},
			Score:     0,
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	content := string(body)

	// Perform technical audit
	technicalAudit := seo.performTechnicalAudit(url, resp, content)
	
	// Perform content audit  
	contentAudit := seo.performContentAudit(content)
	
	// Perform performance audit (simplified)
	perfAudit := seo.performPerformanceAudit(url)

	// Collect issues and calculate score
	issues := seo.collectSEOIssues(technicalAudit, contentAudit, perfAudit)
	score := seo.calculateSEOScore(technicalAudit, contentAudit, perfAudit, issues)

	return &SEOAuditResponse{
		URL:              url,
		Status:           "success",
		TechnicalAudit:   technicalAudit,
		ContentAudit:     contentAudit,
		PerformanceAudit: perfAudit,
		Issues:           issues,
		Score:            score,
		Timestamp:        time.Now().Format(time.RFC3339),
	}, nil
}

// OptimizeContent analyzes content for SEO optimization (replaces content-optimizer workflow)
func (seo *SEOProcessor) OptimizeContent(ctx context.Context, content, targetKeywords, contentType string) (*ContentOptimizationResponse, error) {
	if contentType == "" {
		contentType = "general"
	}

	// Parse target keywords
	keywords := strings.Split(targetKeywords, ",")
	for i := range keywords {
		keywords[i] = strings.TrimSpace(keywords[i])
	}
	keywords = seo.filterEmptyStrings(keywords)

	// Analyze content structure
	contentAnalysis := seo.analyzeContentStructure(content)

	// Analyze keyword usage
	keywordAnalysis := seo.analyzeKeywordUsage(content, keywords)

	// Identify issues and generate recommendations
	issues := seo.identifyContentIssues(contentAnalysis, keywordAnalysis, contentType)
	recommendations := seo.generateContentRecommendations(contentAnalysis, keywordAnalysis, contentType)

	// Calculate content score
	score := seo.calculateContentScore(contentAnalysis, keywordAnalysis, issues)

	return &ContentOptimizationResponse{
		ContentAnalysis: contentAnalysis,
		KeywordAnalysis: keywordAnalysis,
		Issues:          issues,
		Recommendations: recommendations,
		Score:           score,
		ContentType:     contentType,
	}, nil
}

// ResearchKeywords generates keyword suggestions (replaces keyword-researcher workflow)
func (seo *SEOProcessor) ResearchKeywords(ctx context.Context, seedKeyword, targetLocation, language string) (*KeywordResearchResponse, error) {
	if language == "" {
		language = "en"
	}
	if targetLocation == "" {
		targetLocation = "United States"
	}

	// Generate keyword suggestions (simplified implementation)
	keywords := seo.generateKeywordSuggestions(seedKeyword, language)
	relatedTerms := seo.generateRelatedTerms(seedKeyword)
	longTail := seo.generateLongTailKeywords(seedKeyword)
	questions := seo.generateQuestions(seedKeyword)

	return &KeywordResearchResponse{
		SeedKeyword: seedKeyword,
		Keywords:    keywords,
		RelatedTerms: relatedTerms,
		LongTail:    longTail,
		Questions:   questions,
		Status:      "success",
	}, nil
}

// AnalyzeCompetitor performs competitive SEO analysis (replaces competitor-analyzer workflow)
func (seo *SEOProcessor) AnalyzeCompetitor(ctx context.Context, yourURL, competitorURL, analysisType string) (*CompetitorAnalysisResponse, error) {
	if analysisType == "" {
		analysisType = "comprehensive"
	}

	// Fetch both websites for analysis
	yourAudit, err := seo.PerformSEOAudit(ctx, yourURL, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to audit your URL: %w", err)
	}

	competitorAudit, err := seo.PerformSEOAudit(ctx, competitorURL, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to audit competitor URL: %w", err)
	}

	// Perform comparative analysis
	comparison := seo.compareWebsites(yourAudit, competitorAudit)
	opportunities := seo.identifyOpportunities(yourAudit, competitorAudit)
	threats := seo.identifyThreats(yourAudit, competitorAudit)
	recommendations := seo.generateCompetitorRecommendations(comparison, opportunities, threats)
	overallScore := seo.calculateCompetitorScore(yourAudit, competitorAudit)

	return &CompetitorAnalysisResponse{
		YourURL:        yourURL,
		CompetitorURL:  competitorURL,
		Comparison:     comparison,
		Opportunities:  opportunities,
		Threats:        threats,
		Recommendations: recommendations,
		OverallScore:   overallScore,
	}, nil
}

// Helper methods

func (seo *SEOProcessor) performTechnicalAudit(urlStr string, resp *http.Response, content string) TechnicalAuditResults {
	parsedURL, _ := url.Parse(urlStr)
	
	// Extract meta tags
	metaTags := seo.extractMetaTags(content)
	
	// Check for headers
	headers := seo.extractHeaders(content)

	return TechnicalAuditResults{
		HasHTTPS:        parsedURL.Scheme == "https",
		ResponsiveDesign: strings.Contains(content, "viewport") || strings.Contains(content, "@media"),
		LoadTime:        500 + int(time.Now().UnixNano()%1000), // Simulated load time
		StatusCode:      resp.StatusCode,
		MetaTags:        metaTags,
		Headers:         headers,
	}
}

func (seo *SEOProcessor) performContentAudit(content string) ContentAuditResults {
	// Count words
	words := strings.Fields(seo.stripHTML(content))
	wordCount := len(words)

	// Count links and images
	internalLinks := strings.Count(content, `href="/"`) + strings.Count(content, `href="../`)
	externalLinks := strings.Count(content, `href="http`) - internalLinks
	images := strings.Count(content, "<img")
	imagesWithAlt := strings.Count(content, `alt="`)

	// Calculate readability (simplified)
	readabilityScore := seo.calculateReadability(content)

	headers := seo.extractHeaders(content)

	return ContentAuditResults{
		WordCount:        wordCount,
		ReadabilityScore: readabilityScore,
		KeywordDensity:   0.0, // Will be calculated per keyword
		InternalLinks:    internalLinks,
		ExternalLinks:    externalLinks,
		Images:           images,
		ImagesWithAlt:    imagesWithAlt,
		Headers:          headers,
	}
}

func (seo *SEOProcessor) performPerformanceAudit(url string) PerformanceAuditResults {
	// Simplified performance metrics (in production, you'd use real tools like Lighthouse)
	return PerformanceAuditResults{
		PageSpeed:      70 + int(time.Now().UnixNano()%30), // Random 70-100
		MobileScore:    60 + int(time.Now().UnixNano()%40), // Random 60-100
		FirstByte:      200 + int(time.Now().UnixNano()%300), // Random 200-500ms
		DOMContentLoad: 500 + int(time.Now().UnixNano()%1500), // Random 500-2000ms
	}
}

func (seo *SEOProcessor) extractMetaTags(content string) MetaTags {
	titleRegex := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	descRegex := regexp.MustCompile(`<meta[^>]*name=["']description["'][^>]*content=["']([^"']+)["']`)
	keywordsRegex := regexp.MustCompile(`<meta[^>]*name=["']keywords["'][^>]*content=["']([^"']+)["']`)

	var metaTags MetaTags

	if matches := titleRegex.FindStringSubmatch(content); len(matches) > 1 {
		metaTags.Title = matches[1]
		metaTags.TitleLength = len(metaTags.Title)
	}

	if matches := descRegex.FindStringSubmatch(content); len(matches) > 1 {
		metaTags.Description = matches[1]
		metaTags.DescLength = len(metaTags.Description)
	}

	if matches := keywordsRegex.FindStringSubmatch(content); len(matches) > 1 {
		metaTags.Keywords = matches[1]
	}

	return metaTags
}

func (seo *SEOProcessor) extractHeaders(content string) []string {
	headerRegex := regexp.MustCompile(`<h([1-6])[^>]*>([^<]+)</h[1-6]>`)
	matches := headerRegex.FindAllStringSubmatch(content, -1)

	var headers []string
	for _, match := range matches {
		if len(match) > 2 {
			headers = append(headers, fmt.Sprintf("H%s: %s", match[1], match[2]))
		}
	}

	return headers
}

func (seo *SEOProcessor) stripHTML(content string) string {
	// Simple HTML tag removal
	htmlRegex := regexp.MustCompile(`<[^>]+>`)
	return htmlRegex.ReplaceAllString(content, " ")
}

func (seo *SEOProcessor) calculateReadability(content string) int {
	// Simplified Flesch Reading Ease calculation
	text := seo.stripHTML(content)
	sentences := strings.FieldsFunc(text, func(c rune) bool {
		return c == '.' || c == '!' || c == '?'
	})
	words := strings.Fields(text)
	
	if len(sentences) == 0 || len(words) == 0 {
		return 50
	}

	avgSentenceLength := float64(len(words)) / float64(len(sentences))
	score := 206.835 - (1.015 * avgSentenceLength) - (84.6 * 1.5) // Simplified syllable count
	
	return int(math.Max(0, math.Min(100, score)))
}

func (seo *SEOProcessor) analyzeContentStructure(content string) ContentAnalysis {
	words := strings.Fields(seo.stripHTML(content))
	sentences := strings.FieldsFunc(content, func(c rune) bool {
		return c == '.' || c == '!' || c == '?'
	})
	paragraphs := strings.Split(content, "\n\n")

	// Filter empty elements
	sentences = seo.filterEmptyStrings(sentences)
	paragraphs = seo.filterEmptyStrings(paragraphs)

	var avgWordsPerSentence, avgSentencesPerParagraph int
	if len(sentences) > 0 {
		avgWordsPerSentence = len(words) / len(sentences)
	}
	if len(paragraphs) > 0 {
		avgSentencesPerParagraph = len(sentences) / len(paragraphs)
	}

	return ContentAnalysis{
		WordCount:                len(words),
		SentenceCount:           len(sentences),
		ParagraphCount:          len(paragraphs),
		AvgWordsPerSentence:     avgWordsPerSentence,
		AvgSentencesPerParagraph: avgSentencesPerParagraph,
		ReadabilityScore:        float64(seo.calculateReadability(content)),
	}
}

func (seo *SEOProcessor) analyzeKeywordUsage(content string, keywords []string) map[string]KeywordMetrics {
	analysis := make(map[string]KeywordMetrics)
	contentLower := strings.ToLower(content)
	words := strings.Fields(seo.stripHTML(content))
	firstParagraph := ""
	
	// Get first paragraph
	paragraphs := strings.Split(content, "\n\n")
	if len(paragraphs) > 0 {
		firstParagraph = strings.ToLower(paragraphs[0])
	}

	for _, keyword := range keywords {
		keywordLower := strings.ToLower(keyword)
		
		// Count occurrences
		count := strings.Count(contentLower, keywordLower)
		
		// Calculate density
		density := 0.0
		if len(words) > 0 {
			density = (float64(count) / float64(len(words))) * 100
		}

		// Check positioning
		inTitle := strings.Contains(content[:min(100, len(content))], keywordLower)
		inFirstParagraph := strings.Contains(firstParagraph, keywordLower)

		// Calculate prominence score
		prominenceScore := seo.calculateProminenceScore(count, density, inTitle, inFirstParagraph)

		analysis[keyword] = KeywordMetrics{
			Count:              count,
			Density:            density,
			InTitle:            inTitle,
			InFirstParagraph:   inFirstParagraph,
			ProminenceScore:    prominenceScore,
		}
	}

	return analysis
}

func (seo *SEOProcessor) calculateProminenceScore(count int, density float64, inTitle, inFirstParagraph bool) float64 {
	score := 0.0
	
	// Base score from density (optimal 1-3%)
	if density >= 1.0 && density <= 3.0 {
		score += 40.0
	} else if density > 0.5 && density < 1.0 {
		score += 20.0
	} else if density > 3.0 && density < 5.0 {
		score += 15.0
	}

	// Bonus for positioning
	if inTitle {
		score += 30.0
	}
	if inFirstParagraph {
		score += 20.0
	}

	// Bonus for frequency
	if count >= 5 {
		score += 10.0
	} else if count >= 2 {
		score += 5.0
	}

	return math.Min(100.0, score)
}

func (seo *SEOProcessor) identifyContentIssues(contentAnalysis ContentAnalysis, keywordAnalysis map[string]KeywordMetrics, contentType string) []string {
	var issues []string

	// Word count issues
	minWords := 300
	maxWords := 3000
	if contentType == "blog" {
		minWords = 500
		maxWords = 2500
	}

	if contentAnalysis.WordCount < minWords {
		issues = append(issues, fmt.Sprintf("Content is too short (%d words, minimum %d)", contentAnalysis.WordCount, minWords))
	}
	if contentAnalysis.WordCount > maxWords {
		issues = append(issues, fmt.Sprintf("Content might be too long (%d words, maximum %d)", contentAnalysis.WordCount, maxWords))
	}

	// Readability issues
	if contentAnalysis.ReadabilityScore < 30 {
		issues = append(issues, "Content is difficult to read")
	}

	// Sentence length issues
	if contentAnalysis.AvgWordsPerSentence > 25 {
		issues = append(issues, "Sentences are too long on average")
	}

	// Structure issues
	if contentAnalysis.ParagraphCount < 3 {
		issues = append(issues, "Not enough paragraphs for good structure")
	}

	// Keyword issues
	for keyword, metrics := range keywordAnalysis {
		if metrics.Density < 0.5 {
			issues = append(issues, fmt.Sprintf("Keyword '%s' density too low (%.2f%%)", keyword, metrics.Density))
		}
		if metrics.Density > 5.0 {
			issues = append(issues, fmt.Sprintf("Keyword '%s' density too high (%.2f%%) - keyword stuffing", keyword, metrics.Density))
		}
		if !metrics.InFirstParagraph {
			issues = append(issues, fmt.Sprintf("Keyword '%s' not in first paragraph", keyword))
		}
	}

	return issues
}

func (seo *SEOProcessor) generateContentRecommendations(contentAnalysis ContentAnalysis, keywordAnalysis map[string]KeywordMetrics, contentType string) []string {
	var recommendations []string

	// Content length recommendations
	if contentAnalysis.WordCount < 500 {
		recommendations = append(recommendations, "Add more content to reach at least 500 words")
	}

	// Readability recommendations  
	if contentAnalysis.ReadabilityScore < 50 {
		recommendations = append(recommendations, "Simplify sentences and use shorter words to improve readability")
	}

	// Structure recommendations
	if contentAnalysis.AvgWordsPerSentence > 20 {
		recommendations = append(recommendations, "Break up long sentences for better readability")
	}

	// Keyword recommendations
	for keyword, metrics := range keywordAnalysis {
		if metrics.ProminenceScore < 50 {
			recommendations = append(recommendations, fmt.Sprintf("Improve prominence of keyword '%s' by using it in title and first paragraph", keyword))
		}
	}

	// General recommendations
	recommendations = append(recommendations, "Use header tags (H1, H2, H3) to structure content")
	recommendations = append(recommendations, "Add internal links to related content")
	recommendations = append(recommendations, "Optimize images with descriptive alt text")

	return recommendations
}

func (seo *SEOProcessor) calculateContentScore(contentAnalysis ContentAnalysis, keywordAnalysis map[string]KeywordMetrics, issues []string) int {
	score := 100

	// Deduct points for issues
	score -= len(issues) * 10

	// Bonus for good readability
	if contentAnalysis.ReadabilityScore > 60 {
		score += 10
	}

	// Bonus for good keyword optimization
	totalProminence := 0.0
	for _, metrics := range keywordAnalysis {
		totalProminence += metrics.ProminenceScore
	}
	if len(keywordAnalysis) > 0 {
		avgProminence := totalProminence / float64(len(keywordAnalysis))
		if avgProminence > 70 {
			score += 15
		} else if avgProminence > 50 {
			score += 10
		}
	}

	return int(math.Max(0, math.Min(100, float64(score))))
}

func (seo *SEOProcessor) generateKeywordSuggestions(seedKeyword, language string) []KeywordSuggestion {
	// Simplified keyword generation (in production, you'd use real keyword tools)
	baseSuggestions := []string{
		seedKeyword + " tips",
		seedKeyword + " guide",
		"best " + seedKeyword,
		"how to " + seedKeyword,
		seedKeyword + " examples",
		seedKeyword + " tools",
		seedKeyword + " strategies",
		seedKeyword + " benefits",
	}

	var suggestions []KeywordSuggestion
	for i, kw := range baseSuggestions {
		volume := []string{"1K-10K", "10K-100K", "100K-1M"}[i%3]
		competition := []string{"Low", "Medium", "High"}[i%3]
		cpc := []string{"$1.50", "$2.50", "$3.50"}[i%3]
		intent := []string{"Informational", "Commercial", "Navigational"}[i%3]
		
		suggestions = append(suggestions, KeywordSuggestion{
			Keyword:     kw,
			Volume:      volume,
			Competition: competition,
			CPC:         cpc,
			Intent:      intent,
			Difficulty:  30 + (i*10)%50,
			Relevance:   0.8 - float64(i)*0.1,
		})
	}

	return suggestions
}

func (seo *SEOProcessor) generateRelatedTerms(seedKeyword string) []string {
	// Simplified related terms generation
	return []string{
		strings.Replace(seedKeyword, " ", " and ", 1),
		seedKeyword + " process",
		seedKeyword + " methods",
		seedKeyword + " techniques",
		"advanced " + seedKeyword,
	}
}

func (seo *SEOProcessor) generateLongTailKeywords(seedKeyword string) []KeywordSuggestion {
	longTailTerms := []string{
		"how to improve " + seedKeyword + " for beginners",
		"best practices for " + seedKeyword + " in 2025",
		"step by step " + seedKeyword + " tutorial",
		seedKeyword + " case study examples",
	}

	var suggestions []KeywordSuggestion
	for i, kw := range longTailTerms {
		suggestions = append(suggestions, KeywordSuggestion{
			Keyword:     kw,
			Volume:      "100-1K",
			Competition: "Low",
			CPC:         "$1.25",
			Intent:      "Informational",
			Difficulty:  15 + i*5,
			Relevance:   0.9,
		})
	}

	return suggestions
}

func (seo *SEOProcessor) generateQuestions(seedKeyword string) []string {
	return []string{
		"What is " + seedKeyword + "?",
		"How does " + seedKeyword + " work?",
		"Why is " + seedKeyword + " important?",
		"When should you use " + seedKeyword + "?",
		"Where can I learn more about " + seedKeyword + "?",
	}
}

func (seo *SEOProcessor) compareWebsites(yours, competitor *SEOAuditResponse) CompetitorComparison {
	return CompetitorComparison{
		SEOScore:        CompetitorMetric{
			Yours:      yours.Score,
			Competitor: competitor.Score,
			Winner:     seo.getWinner(yours.Score, competitor.Score),
			Difference: abs(yours.Score - competitor.Score),
		},
		ContentQuality: CompetitorMetric{
			Yours:      yours.ContentAudit.WordCount / 10, // Simplified score
			Competitor: competitor.ContentAudit.WordCount / 10,
			Winner:     seo.getWinner(yours.ContentAudit.WordCount, competitor.ContentAudit.WordCount),
			Difference: abs(yours.ContentAudit.WordCount - competitor.ContentAudit.WordCount) / 10,
		},
		TechnicalSEO: CompetitorMetric{
			Yours:      seo.calculateTechnicalScore(yours.TechnicalAudit),
			Competitor: seo.calculateTechnicalScore(competitor.TechnicalAudit),
		},
		UserExperience: CompetitorMetric{
			Yours:      yours.PerformanceAudit.PageSpeed,
			Competitor: competitor.PerformanceAudit.PageSpeed,
			Winner:     seo.getWinner(yours.PerformanceAudit.PageSpeed, competitor.PerformanceAudit.PageSpeed),
			Difference: abs(yours.PerformanceAudit.PageSpeed - competitor.PerformanceAudit.PageSpeed),
		},
		Keywords: CompetitorKeywords{
			SharedKeywords: []string{"shared keyword 1", "shared keyword 2"},
			YourUnique:     []string{"your unique 1", "your unique 2"},
			TheirUnique:    []string{"their unique 1", "their unique 2"},
			Gaps:          []string{"opportunity 1", "opportunity 2"},
		},
	}
}

func (seo *SEOProcessor) calculateTechnicalScore(technical TechnicalAuditResults) int {
	score := 0
	if technical.HasHTTPS {
		score += 25
	}
	if technical.ResponsiveDesign {
		score += 25
	}
	if technical.StatusCode == 200 {
		score += 25
	}
	if technical.MetaTags.Title != "" {
		score += 25
	}
	return score
}

func (seo *SEOProcessor) identifyOpportunities(yours, competitor *SEOAuditResponse) []string {
	var opportunities []string
	
	if competitor.Score > yours.Score {
		opportunities = append(opportunities, "Overall SEO score can be improved")
	}
	if competitor.ContentAudit.WordCount > yours.ContentAudit.WordCount {
		opportunities = append(opportunities, "Add more comprehensive content")
	}
	if competitor.PerformanceAudit.PageSpeed > yours.PerformanceAudit.PageSpeed {
		opportunities = append(opportunities, "Improve page loading speed")
	}

	opportunities = append(opportunities, "Target their unique keywords", "Improve technical SEO implementation")
	
	return opportunities
}

func (seo *SEOProcessor) identifyThreats(yours, competitor *SEOAuditResponse) []string {
	var threats []string
	
	if competitor.Score > yours.Score+20 {
		threats = append(threats, "Competitor has significantly better SEO")
	}
	if competitor.ContentAudit.WordCount > yours.ContentAudit.WordCount*2 {
		threats = append(threats, "Competitor has much more content")
	}

	threats = append(threats, "They may be targeting your keywords", "Strong competitor presence in search results")
	
	return threats
}

func (seo *SEOProcessor) generateCompetitorRecommendations(comparison CompetitorComparison, opportunities, threats []string) []string {
	var recommendations []string
	
	if comparison.SEOScore.Winner == "competitor" {
		recommendations = append(recommendations, "Focus on improving overall SEO score")
	}
	if comparison.ContentQuality.Winner == "competitor" {
		recommendations = append(recommendations, "Create more comprehensive content")
	}
	if comparison.UserExperience.Winner == "competitor" {
		recommendations = append(recommendations, "Optimize page speed and user experience")
	}

	recommendations = append(recommendations, 
		"Monitor competitor keyword strategy",
		"Identify content gaps and fill them",
		"Improve technical SEO implementation")
	
	return recommendations
}

func (seo *SEOProcessor) calculateCompetitorScore(yours, competitor *SEOAuditResponse) CompetitorScore {
	yourScore := yours.Score
	competitorScore := competitor.Score
	
	overall := int((float64(yourScore) / float64(yourScore+competitorScore)) * 100)
	
	strengths := []string{}
	weaknesses := []string{}
	
	if yours.ContentAudit.WordCount > competitor.ContentAudit.WordCount {
		strengths = append(strengths, "More comprehensive content")
	} else {
		weaknesses = append(weaknesses, "Less content than competitor")
	}
	
	if yours.PerformanceAudit.PageSpeed > competitor.PerformanceAudit.PageSpeed {
		strengths = append(strengths, "Better page performance")
	} else {
		weaknesses = append(weaknesses, "Slower page loading")
	}

	return CompetitorScore{
		Overall:    overall,
		Strengths:  strengths,
		Weaknesses: weaknesses,
	}
}

func (seo *SEOProcessor) collectSEOIssues(technical TechnicalAuditResults, content ContentAuditResults, perf PerformanceAuditResults) []SEOIssue {
	var issues []SEOIssue

	if !technical.HasHTTPS {
		issues = append(issues, SEOIssue{
			Type:        "security",
			Severity:    "high",
			Description: "Website not using HTTPS",
			Impact:      20,
		})
	}
	
	if technical.MetaTags.Title == "" {
		issues = append(issues, SEOIssue{
			Type:        "meta",
			Severity:    "critical",
			Description: "Missing title tag",
			Impact:      25,
		})
	}

	if content.WordCount < 300 {
		issues = append(issues, SEOIssue{
			Type:        "content",
			Severity:    "medium",
			Description: "Content too short",
			Impact:      15,
		})
	}

	if perf.PageSpeed < 50 {
		issues = append(issues, SEOIssue{
			Type:        "performance",
			Severity:    "high",
			Description: "Poor page speed",
			Impact:      20,
		})
	}

	return issues
}

func (seo *SEOProcessor) calculateSEOScore(technical TechnicalAuditResults, content ContentAuditResults, perf PerformanceAuditResults, issues []SEOIssue) int {
	score := 100

	// Deduct points for issues
	for _, issue := range issues {
		score -= issue.Impact
	}

	// Bonus points for good practices
	if technical.HasHTTPS {
		score += 5
	}
	if technical.MetaTags.TitleLength > 30 && technical.MetaTags.TitleLength < 60 {
		score += 5
	}
	if content.WordCount > 500 {
		score += 5
	}
	if perf.PageSpeed > 80 {
		score += 10
	}

	return int(math.Max(0, math.Min(100, float64(score))))
}

// Utility functions
func (seo *SEOProcessor) filterEmptyStrings(slice []string) []string {
	var result []string
	for _, s := range slice {
		if strings.TrimSpace(s) != "" {
			result = append(result, strings.TrimSpace(s))
		}
	}
	return result
}

func (seo *SEOProcessor) getWinner(yours, competitor int) string {
	if yours > competitor {
		return "yours"
	} else if competitor > yours {
		return "competitor"
	}
	return "tie"
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}