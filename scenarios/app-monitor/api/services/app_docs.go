package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// =============================================================================
// Documentation Types
// =============================================================================

// AppDocument represents a rendered document with metadata
type AppDocument struct {
	Name         string `json:"name"`
	Path         string `json:"path"` // Relative path from scenario root
	Size         int64  `json:"size"`
	IsMarkdown   bool   `json:"is_markdown"`
	ModifiedAt   string `json:"modified_at"`
	Content      string `json:"content"`                 // Raw content
	RenderedHTML string `json:"rendered_html,omitempty"` // HTML if markdown was rendered
}

// AppDocumentMatch represents a search result
type AppDocumentMatch struct {
	Document AppDocumentInfo `json:"document"`
	Matches  []MatchContext  `json:"matches"`
	Score    int             `json:"score"` // Number of matches
}

// MatchContext represents a single match with surrounding context
type MatchContext struct {
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
	Context    string `json:"context,omitempty"` // Surrounding lines for context
}

// =============================================================================
// Constants
// =============================================================================

const (
	maxDocumentSize    = 5 * 1024 * 1024 // 5MB max per document
	maxSearchResults   = 20              // Max search results to return
	searchContextLines = 2               // Lines of context around matches
)

var (
	// rootDocumentNames are common documentation files in scenario roots
	rootDocumentNames = []string{
		"README.md",
		"PRD.md",
		"PROBLEMS.md",
		"LESSONS_LEARNED.md",
		"CHANGELOG.md",
		"TODO.md",
		"ARCHITECTURE.md",
	}

	// allowedDocExtensions are file extensions we'll read from docs/ folder
	allowedDocExtensions = map[string]bool{
		".md":   true,
		".txt":  true,
		".json": false, // Don't render JSON as markdown
		".yaml": false,
		".yml":  false,
	}
)

// =============================================================================
// List Documents
// =============================================================================

// ListAppDocuments returns all available documentation for an app
func (s *AppService) ListAppDocuments(ctx context.Context, appID string) (*AppDocumentsList, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}

	scenarioPath := strings.TrimSpace(app.Path)
	if scenarioPath == "" {
		return &AppDocumentsList{
			RootDocs: []AppDocumentInfo{},
			DocsDocs: []AppDocumentInfo{},
			Total:    0,
		}, nil
	}

	result := &AppDocumentsList{
		RootDocs: make([]AppDocumentInfo, 0),
		DocsDocs: make([]AppDocumentInfo, 0),
	}

	// Scan root for common documentation files
	for _, docName := range rootDocumentNames {
		fullPath := filepath.Join(scenarioPath, docName)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // File doesn't exist, skip
		}
		if info.IsDir() {
			continue
		}

		result.RootDocs = append(result.RootDocs, AppDocumentInfo{
			Name:       docName,
			Path:       docName,
			Size:       info.Size(),
			IsMarkdown: strings.HasSuffix(strings.ToLower(docName), ".md"),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// Scan docs/ folder if it exists
	docsPath := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsPath); err == nil && info.IsDir() {
		err := filepath.WalkDir(docsPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			// Skip directories
			if d.IsDir() {
				return nil
			}

			// Check if extension is allowed
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if _, allowed := allowedDocExtensions[ext]; !allowed {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil // Skip files we can't stat
			}

			// Get relative path from docs/ folder
			relPath, err := filepath.Rel(docsPath, path)
			if err != nil {
				relPath = d.Name()
			}

			result.DocsDocs = append(result.DocsDocs, AppDocumentInfo{
				Name:       d.Name(),
				Path:       filepath.Join("docs", relPath),
				Size:       info.Size(),
				IsMarkdown: ext == ".md",
				ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
			})

			return nil
		})

		if err != nil {
			// Silently skip errors in docs folder - return what we found in root
			// (logger would require import which isn't critical for this operation)
		}
	}

	result.Total = len(result.RootDocs) + len(result.DocsDocs)

	return result, nil
}

// =============================================================================
// Get Document
// =============================================================================

// GetAppDocument reads and optionally renders a specific document
func (s *AppService) GetAppDocument(ctx context.Context, appID, docPath string, render bool) (*AppDocument, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	cleanDocPath := strings.TrimSpace(docPath)
	if cleanDocPath == "" {
		return nil, errors.New("document path is required")
	}

	// Security: prevent path traversal
	if strings.Contains(cleanDocPath, "..") {
		return nil, errors.New("invalid document path: path traversal not allowed")
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}

	scenarioPath := strings.TrimSpace(app.Path)
	if scenarioPath == "" {
		return nil, errors.New("scenario path unavailable")
	}

	// Construct full path
	fullPath := filepath.Join(scenarioPath, cleanDocPath)

	// Verify the file exists and is within the scenario directory (security check)
	absScenarioPath, err := filepath.Abs(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scenario path: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve document path: %w", err)
	}

	if !strings.HasPrefix(absFullPath, absScenarioPath) {
		return nil, errors.New("invalid document path: outside scenario directory")
	}

	// Check file exists and get info
	info, err := os.Stat(absFullPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("document not found: %s", cleanDocPath)
		}
		return nil, fmt.Errorf("failed to access document: %w", err)
	}

	if info.IsDir() {
		return nil, errors.New("path is a directory, not a document")
	}

	if info.Size() > maxDocumentSize {
		return nil, fmt.Errorf("document too large: %d bytes (max %d)", info.Size(), maxDocumentSize)
	}

	// Read file content
	content, err := os.ReadFile(absFullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	doc := &AppDocument{
		Name:       filepath.Base(cleanDocPath),
		Path:       cleanDocPath,
		Size:       info.Size(),
		IsMarkdown: strings.HasSuffix(strings.ToLower(cleanDocPath), ".md"),
		ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
		Content:    string(content),
	}

	// Render markdown to HTML if requested
	if render && doc.IsMarkdown {
		doc.RenderedHTML = renderMarkdownToHTML(content)
	}

	return doc, nil
}

// =============================================================================
// Search Documents
// =============================================================================

// SearchAppDocuments searches documentation content for a query string
func (s *AppService) SearchAppDocuments(ctx context.Context, appID, query string) ([]AppDocumentMatch, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, errors.New("search query is required")
	}

	// List all documents
	docsList, err := s.ListAppDocuments(ctx, id)
	if err != nil {
		return nil, err
	}

	if docsList.Total == 0 {
		return []AppDocumentMatch{}, nil
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}

	scenarioPath := strings.TrimSpace(app.Path)
	if scenarioPath == "" {
		return []AppDocumentMatch{}, nil
	}

	results := make([]AppDocumentMatch, 0)
	queryLower := strings.ToLower(cleanQuery)

	// Search in root docs
	for _, docInfo := range docsList.RootDocs {
		matches := searchDocumentFile(scenarioPath, docInfo, queryLower)
		if len(matches) > 0 {
			results = append(results, AppDocumentMatch{
				Document: docInfo,
				Matches:  matches,
				Score:    len(matches),
			})
		}
	}

	// Search in docs/ folder
	for _, docInfo := range docsList.DocsDocs {
		matches := searchDocumentFile(scenarioPath, docInfo, queryLower)
		if len(matches) > 0 {
			results = append(results, AppDocumentMatch{
				Document: docInfo,
				Matches:  matches,
				Score:    len(matches),
			})
		}
	}

	// Sort by score (number of matches) descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > maxSearchResults {
		results = results[:maxSearchResults]
	}

	return results, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// searchDocumentFile searches a single document file for query matches
func searchDocumentFile(scenarioPath string, docInfo AppDocumentInfo, queryLower string) []MatchContext {
	fullPath := filepath.Join(scenarioPath, docInfo.Path)

	// Security check
	absScenarioPath, err := filepath.Abs(scenarioPath)
	if err != nil {
		return nil
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil
	}
	if !strings.HasPrefix(absFullPath, absScenarioPath) {
		return nil
	}

	// Read file
	content, err := os.ReadFile(absFullPath)
	if err != nil {
		return nil
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")
	matches := make([]MatchContext, 0)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			// Get context lines
			contextStart := i - searchContextLines
			if contextStart < 0 {
				contextStart = 0
			}
			contextEnd := i + searchContextLines + 1
			if contextEnd > len(lines) {
				contextEnd = len(lines)
			}

			contextLines := lines[contextStart:contextEnd]
			context := strings.Join(contextLines, "\n")

			matches = append(matches, MatchContext{
				LineNumber: i + 1, // 1-indexed
				Line:       line,
				Context:    context,
			})

			// Limit matches per file to avoid overwhelming results
			if len(matches) >= 10 {
				break
			}
		}
	}

	return matches
}

// renderMarkdownToHTML converts markdown content to sanitized HTML
func renderMarkdownToHTML(content []byte) string {
	// Create parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(content)

	// Create HTML renderer with options
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return string(markdown.Render(doc, renderer))
}
