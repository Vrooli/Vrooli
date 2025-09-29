package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"
	
	"github.com/google/uuid"
)

// ExportFormat represents the export format type
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatMD   ExportFormat = "markdown"
)

// ResearchExport contains exported research data
type ResearchExport struct {
	ExportID        string                 `json:"export_id"`
	ExportDate      time.Time              `json:"export_date"`
	Description     string                 `json:"description"`
	InjectionTechniques []InjectionTechnique `json:"injection_techniques"`
	TestResults     []TestResult           `json:"test_results"`
	Statistics      map[string]interface{} `json:"statistics"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ExportResearchData exports research data in the specified format
func ExportResearchData(format ExportFormat, filters map[string]interface{}) ([]byte, error) {
	export := ResearchExport{
		ExportID:   uuid.New().String(),
		ExportDate: time.Now(),
		Description: "Prompt Injection Arena Research Export",
		Metadata: map[string]interface{}{
			"version": "1.0",
			"source":  "prompt-injection-arena",
			"filters": filters,
		},
	}
	
	// Get injection techniques
	techniques, err := getFilteredInjectionTechniques(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get injection techniques: %v", err)
	}
	export.InjectionTechniques = techniques
	
	// Get test results
	results, err := getFilteredTestResults(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get test results: %v", err)
	}
	export.TestResults = results
	
	// Calculate statistics
	export.Statistics = calculateExportStatistics(techniques, results)
	
	// Export based on format
	switch format {
	case FormatJSON:
		return exportAsJSON(export)
	case FormatCSV:
		return exportAsCSV(export)
	case FormatMD:
		return exportAsMarkdown(export)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// getFilteredInjectionTechniques retrieves injection techniques based on filters
func getFilteredInjectionTechniques(filters map[string]interface{}) ([]InjectionTechnique, error) {
	query := `SELECT id, name, category, description, example_prompt, difficulty_score, 
	          success_rate, source_attribution, is_active, created_at, updated_at, created_by
	          FROM injection_techniques WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	// Apply filters
	if category, ok := filters["category"].(string); ok && category != "" {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, category)
	}
	
	if minDifficulty, ok := filters["min_difficulty"].(float64); ok {
		argCount++
		query += fmt.Sprintf(" AND difficulty_score >= $%d", argCount)
		args = append(args, minDifficulty)
	}
	
	if minSuccessRate, ok := filters["min_success_rate"].(float64); ok {
		argCount++
		query += fmt.Sprintf(" AND success_rate >= $%d", argCount)
		args = append(args, minSuccessRate)
	}
	
	query += " ORDER BY category, difficulty_score DESC"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var techniques []InjectionTechnique
	for rows.Next() {
		var tech InjectionTechnique
		err := rows.Scan(
			&tech.ID, &tech.Name, &tech.Category, &tech.Description,
			&tech.ExamplePrompt, &tech.DifficultyScore, &tech.SuccessRate,
			&tech.SourceAttribution, &tech.IsActive, &tech.CreatedAt,
			&tech.UpdatedAt, &tech.CreatedBy,
		)
		if err == nil {
			techniques = append(techniques, tech)
		}
	}
	
	return techniques, nil
}

// getFilteredTestResults retrieves test results based on filters
func getFilteredTestResults(filters map[string]interface{}) ([]TestResult, error) {
	query := `SELECT id, injection_id, agent_id, success, response_text, execution_time_ms,
	          confidence_score, error_message, executed_at, test_session_id
	          FROM test_results WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	// Apply date range filter
	if startDate, ok := filters["start_date"].(time.Time); ok {
		argCount++
		query += fmt.Sprintf(" AND executed_at >= $%d", argCount)
		args = append(args, startDate)
	}
	
	if endDate, ok := filters["end_date"].(time.Time); ok {
		argCount++
		query += fmt.Sprintf(" AND executed_at <= $%d", argCount)
		args = append(args, endDate)
	}
	
	// Limit results for export
	limit := 1000
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	argCount++
	query += fmt.Sprintf(" ORDER BY executed_at DESC LIMIT $%d", argCount)
	args = append(args, limit)
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []TestResult
	for rows.Next() {
		var result TestResult
		err := rows.Scan(
			&result.ID, &result.InjectionID, &result.AgentID, &result.Success,
			&result.ResponseText, &result.ExecutionTimeMS, &result.ConfidenceScore,
			&result.ErrorMessage, &result.ExecutedAt, &result.TestSessionID,
		)
		if err == nil {
			// Truncate response text for export
			if len(result.ResponseText) > 500 {
				result.ResponseText = result.ResponseText[:500] + "..."
			}
			results = append(results, result)
		}
	}
	
	return results, nil
}

// calculateExportStatistics calculates statistics for the export
func calculateExportStatistics(techniques []InjectionTechnique, results []TestResult) map[string]interface{} {
	stats := map[string]interface{}{}
	
	// Technique statistics
	categoryCount := make(map[string]int)
	totalDifficulty := 0.0
	totalSuccessRate := 0.0
	
	for _, tech := range techniques {
		categoryCount[tech.Category]++
		totalDifficulty += tech.DifficultyScore
		totalSuccessRate += tech.SuccessRate
	}
	
	stats["total_techniques"] = len(techniques)
	stats["categories"] = categoryCount
	if len(techniques) > 0 {
		stats["avg_difficulty"] = totalDifficulty / float64(len(techniques))
		stats["avg_success_rate"] = totalSuccessRate / float64(len(techniques))
	}
	
	// Result statistics
	successCount := 0
	totalExecutionTime := 0
	totalConfidence := 0.0
	
	for _, result := range results {
		if result.Success {
			successCount++
		}
		totalExecutionTime += result.ExecutionTimeMS
		totalConfidence += result.ConfidenceScore
	}
	
	stats["total_tests"] = len(results)
	stats["successful_injections"] = successCount
	if len(results) > 0 {
		stats["success_percentage"] = float64(successCount) / float64(len(results)) * 100
		stats["avg_execution_time_ms"] = totalExecutionTime / len(results)
		stats["avg_confidence_score"] = totalConfidence / float64(len(results))
	}
	
	return stats
}

// exportAsJSON exports data as JSON
func exportAsJSON(export ResearchExport) ([]byte, error) {
	return json.MarshalIndent(export, "", "  ")
}

// exportAsCSV exports injection techniques as CSV
func exportAsCSV(export ResearchExport) ([]byte, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "export-*.csv")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	
	writer := csv.NewWriter(tmpFile)
	defer writer.Flush()
	
	// Write header
	header := []string{"ID", "Name", "Category", "Description", "Example Prompt", 
		"Difficulty Score", "Success Rate", "Source Attribution", "Created At"}
	writer.Write(header)
	
	// Write data
	for _, tech := range export.InjectionTechniques {
		record := []string{
			tech.ID,
			tech.Name,
			tech.Category,
			tech.Description,
			tech.ExamplePrompt,
			fmt.Sprintf("%.2f", tech.DifficultyScore),
			fmt.Sprintf("%.2f%%", tech.SuccessRate*100),
			tech.SourceAttribution,
			tech.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		writer.Write(record)
	}
	
	writer.Flush()
	
	// Read file contents
	tmpFile.Seek(0, 0)
	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := tmpFile.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	
	return data, nil
}

// exportAsMarkdown exports data as Markdown report
func exportAsMarkdown(export ResearchExport) ([]byte, error) {
	md := fmt.Sprintf(`# Prompt Injection Arena Research Export

**Export ID:** %s  
**Export Date:** %s  
**Description:** %s

## Executive Summary

This export contains **%d injection techniques** across **%d categories** with an average difficulty score of **%.2f** and an average success rate of **%.1f%%**.

## Statistics

`, export.ExportID, export.ExportDate.Format("2006-01-02 15:04:05"), 
		export.Description, len(export.InjectionTechniques),
		len(export.Statistics["categories"].(map[string]int)),
		export.Statistics["avg_difficulty"].(float64),
		export.Statistics["avg_success_rate"].(float64)*100)
	
	// Add statistics section
	md += "### Injection Technique Statistics\n\n"
	md += fmt.Sprintf("- Total Techniques: %d\n", export.Statistics["total_techniques"])
	md += fmt.Sprintf("- Average Difficulty: %.2f\n", export.Statistics["avg_difficulty"])
	md += fmt.Sprintf("- Average Success Rate: %.1f%%\n\n", export.Statistics["avg_success_rate"].(float64)*100)
	
	md += "### Category Distribution\n\n"
	for category, count := range export.Statistics["categories"].(map[string]int) {
		md += fmt.Sprintf("- %s: %d techniques\n", category, count)
	}
	
	md += "\n### Test Results Statistics\n\n"
	md += fmt.Sprintf("- Total Tests: %d\n", export.Statistics["total_tests"])
	md += fmt.Sprintf("- Successful Injections: %d\n", export.Statistics["successful_injections"])
	if totalTests, ok := export.Statistics["total_tests"].(int); ok && totalTests > 0 {
		md += fmt.Sprintf("- Success Percentage: %.1f%%\n", export.Statistics["success_percentage"])
		md += fmt.Sprintf("- Average Execution Time: %dms\n", export.Statistics["avg_execution_time_ms"])
		md += fmt.Sprintf("- Average Confidence Score: %.2f\n", export.Statistics["avg_confidence_score"])
	}
	
	// Add injection techniques section
	md += "\n## Injection Techniques\n\n"
	
	// Group by category
	techniquesByCategory := make(map[string][]InjectionTechnique)
	for _, tech := range export.InjectionTechniques {
		techniquesByCategory[tech.Category] = append(techniquesByCategory[tech.Category], tech)
	}
	
	for category, techs := range techniquesByCategory {
		md += fmt.Sprintf("### %s\n\n", category)
		
		for _, tech := range techs {
			md += fmt.Sprintf("#### %s\n", tech.Name)
			md += fmt.Sprintf("- **Difficulty:** %.2f\n", tech.DifficultyScore)
			md += fmt.Sprintf("- **Success Rate:** %.1f%%\n", tech.SuccessRate*100)
			md += fmt.Sprintf("- **Description:** %s\n", tech.Description)
			md += fmt.Sprintf("- **Example:** `%s`\n\n", tech.ExamplePrompt)
		}
	}
	
	// Add responsible disclosure note
	md += `## Responsible Disclosure

This research export is provided for security research and defensive purposes only. The injection techniques documented here should be used to:

1. Test and improve AI system robustness
2. Develop better defensive mechanisms
3. Advance the field of AI safety research

**Do not use these techniques for malicious purposes or unauthorized testing.**

## Metadata

`
	
	metadataJSON, _ := json.MarshalIndent(export.Metadata, "", "  ")
	md += "```json\n" + string(metadataJSON) + "\n```\n"
	
	return []byte(md), nil
}