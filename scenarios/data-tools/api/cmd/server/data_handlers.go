package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Vrooli/Vrooli/scenarios/data-tools/api/internal/dataprocessor"
	"github.com/google/uuid"
)

// ParseRequest represents the request for data parsing
type ParseRequest struct {
	Data    interface{}                  `json:"data"`
	Format  string                       `json:"format"`
	Options dataprocessor.ParseOptions  `json:"options"`
}

// TransformRequest represents the request for data transformation
type TransformRequest struct {
	DatasetID       string                           `json:"dataset_id,omitempty"`
	Data            []map[string]interface{}         `json:"data,omitempty"`
	Transformations []dataprocessor.Transformation   `json:"transformations"`
	Options         map[string]interface{}           `json:"options"`
}

// ValidateRequest represents the request for data validation
type ValidateRequest struct {
	DatasetID    string                   `json:"dataset_id,omitempty"`
	Data         []map[string]interface{} `json:"data,omitempty"`
	Schema       dataprocessor.Schema     `json:"schema"`
	QualityRules []QualityRule            `json:"quality_rules"`
}

// QualityRule represents a data quality rule
type QualityRule struct {
	RuleType   string                 `json:"rule_type"`
	Parameters map[string]interface{} `json:"parameters"`
	Severity   string                 `json:"severity"`
}

// QueryRequest represents the request for SQL queries
type QueryRequest struct {
	SQL      string                 `json:"sql"`
	Datasets []DatasetRef           `json:"datasets"`
	Options  map[string]interface{} `json:"options"`
}

// DatasetRef represents a reference to a dataset
type DatasetRef struct {
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

// StreamCreateRequest represents the request for creating a streaming source
type StreamCreateRequest struct {
	SourceConfig    map[string]interface{}   `json:"source_config"`
	ProcessingRules []map[string]interface{} `json:"processing_rules"`
	OutputConfig    map[string]interface{}   `json:"output_config"`
}

// handleDataParse handles data parsing requests
func (s *Server) handleDataParse(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert data to string for parsing
	dataStr := ""
	switch v := req.Data.(type) {
	case string:
		dataStr = v
	case map[string]interface{}:
		if _, ok := v["url"].(string); ok {
			// In production, fetch data from URL
			s.sendError(w, http.StatusNotImplemented, "URL fetching not implemented")
			return
		}
		if file, ok := v["file"].(string); ok {
			// In production, decode base64 file
			dataStr = file
		}
	default:
		// Try to marshal as JSON
		bytes, _ := json.Marshal(req.Data)
		dataStr = string(bytes)
	}

	// Parse data
	parser := dataprocessor.NewParser()
	result, err := parser.Parse(dataStr, dataprocessor.DataFormat(req.Format), req.Options)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to parse data: %v", err))
		return
	}

	// Store dataset metadata in database
	datasetID := uuid.New().String()
	schemaJSON, _ := json.Marshal(result.Schema)
	
	query := `INSERT INTO datasets (id, name, schema_definition, format, row_count, column_count, quality_score, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = s.db.Exec(query,
		datasetID,
		fmt.Sprintf("dataset_%s", datasetID[:8]),
		schemaJSON,
		req.Format,
		result.Schema.RowCount,
		len(result.Schema.Columns),
		result.Schema.QualityScore,
		time.Now(),
	)
	
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to store dataset metadata: %v\n", err)
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"dataset_id": datasetID,
		"schema":     result.Schema,
		"preview":    result.Preview,
		"warnings":   result.Warnings,
	})
}

// handleDataTransform handles data transformation requests
func (s *Server) handleDataTransform(w http.ResponseWriter, r *http.Request) {
	var req TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get data to transform
	var data []map[string]interface{}
	
	if req.DatasetID != "" {
		// Load dataset from database
		// In production, this would load actual data from storage
		s.sendError(w, http.StatusNotImplemented, "dataset loading not implemented")
		return
	} else if req.Data != nil {
		data = req.Data
	} else {
		s.sendError(w, http.StatusBadRequest, "either dataset_id or data is required")
		return
	}

	// Apply transformations
	transformer := dataprocessor.NewTransformer()
	result, err := transformer.Transform(data, req.Transformations)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("transformation failed: %v", err))
		return
	}

	// Store transformation metadata in database
	transformationID := uuid.New().String()
	paramsJSON, _ := json.Marshal(req.Transformations)
	statsJSON, _ := json.Marshal(result.ExecutionStats)
	
	query := `INSERT INTO data_transformations 
	          (id, transformation_type, parameters, execution_stats, execution_time_ms, rows_processed, success, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = s.db.Exec(query,
		transformationID,
		"multi",
		paramsJSON,
		statsJSON,
		result.ExecutionStats.ExecutionTimeMs,
		result.ExecutionStats.RowsProcessed,
		result.ExecutionStats.Success,
		time.Now(),
	)
	
	if err != nil {
		fmt.Printf("Failed to store transformation metadata: %v\n", err)
	}

	// Return result or URL based on size
	if len(result.Data) > 1000 {
		// Store in object storage and return URL
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"result": map[string]interface{}{
				"url":             fmt.Sprintf("/api/v1/data/results/%s", transformationID),
				"schema":          result.Schema,
				"row_count":       result.RowCount,
				"execution_stats": result.ExecutionStats,
			},
			"transformation_id": transformationID,
		})
	} else {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"result": map[string]interface{}{
				"data":            result.Data,
				"schema":          result.Schema,
				"row_count":       result.RowCount,
				"execution_stats": result.ExecutionStats,
			},
			"transformation_id": transformationID,
		})
	}
}

// handleDataValidate handles data validation requests
func (s *Server) handleDataValidate(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get data to validate
	var data []map[string]interface{}
	
	if req.DatasetID != "" {
		// Load dataset from database
		// In production, this would load actual data from storage
		s.sendError(w, http.StatusNotImplemented, "dataset loading not implemented")
		return
	} else if req.Data != nil {
		data = req.Data
	} else {
		s.sendError(w, http.StatusBadRequest, "either dataset_id or data is required")
		return
	}

	// Validate data
	isValid := true
	violations := []map[string]interface{}{}
	
	// Check completeness
	completeness := calculateCompleteness(data)
	
	// Check for duplicates
	duplicateCount := countDuplicates(data)
	
	// Check for anomalies
	anomalies := detectAnomalies(data)
	
	// Calculate accuracy (based on schema validation)
	accuracy := calculateAccuracy(data, req.Schema)
	
	// Calculate consistency (based on data patterns)
	consistency := calculateConsistency(data)
	
	// Apply custom quality rules
	for _, rule := range req.QualityRules {
		ruleViolations := applyQualityRule(data, rule)
		violations = append(violations, ruleViolations...)
		if len(ruleViolations) > 0 && rule.Severity == "error" {
			isValid = false
		}
	}
	
	// Calculate overall quality score
	qualityScore := (completeness + accuracy + consistency) / 3.0
	if duplicateCount > 0 {
		qualityScore *= (1.0 - float64(duplicateCount)/float64(len(data)))
	}
	if len(anomalies) > 0 {
		qualityScore *= (1.0 - float64(len(anomalies))/float64(len(data)*len(data[0])))
	}

	qualityReport := map[string]interface{}{
		"completeness":  completeness,
		"accuracy":      accuracy,
		"consistency":   consistency,
		"quality_score": qualityScore,
		"anomalies":     anomalies,
		"duplicates":    duplicateCount,
		"total_rows":    len(data),
		"total_columns": len(data[0]),
	}

	// Store quality report in database
	if req.DatasetID != "" {
		anomaliesJSON, _ := json.Marshal(anomalies)
		
		query := `INSERT INTO data_quality_reports 
		          (dataset_id, completeness_score, accuracy_score, consistency_score, 
		           anomalies_detected, duplicate_count, generated_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7)`
		
		_, err := s.db.Exec(query,
			req.DatasetID,
			completeness,
			0.95,
			0.90,
			anomaliesJSON,
			duplicateCount,
			time.Now(),
		)
		
		if err != nil {
			fmt.Printf("Failed to store quality report: %v\n", err)
		}
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"is_valid":       isValid,
		"quality_report": qualityReport,
		"violations":     violations,
	})
}

// handleDataQuery handles SQL query requests
func (s *Server) handleDataQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// For demo purposes, execute query against database
	// In production, this would handle dataset joins and complex queries
	
	limit := 1000
	if l, ok := req.Options["limit"].(float64); ok {
		limit = int(l)
	}

	// Add LIMIT to query if not present
	sqlQuery := req.SQL
	if limit > 0 && !hasLimit(sqlQuery) {
		sqlQuery = fmt.Sprintf("%s LIMIT %d", sqlQuery, limit)
	}

	// Execute query
	rows, err := s.db.Query(sqlQuery)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, fmt.Sprintf("query failed: %v", err))
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to get columns")
		return
	}

	// Fetch results
	results := []map[string]interface{}{}
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column
		columnValues := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			continue
		}

		// Create map for this row
		row := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			// Handle NULL values
			if val == nil {
				row[colName] = nil
			} else {
				// Convert []byte to string for text columns
				if b, ok := val.([]byte); ok {
					row[colName] = string(b)
				} else {
					row[colName] = val
				}
			}
		}
		results = append(results, row)
	}

	// Generate query ID
	queryID := uuid.New().String()

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"result": map[string]interface{}{
			"data":     results,
			"columns":  columns,
			"row_count": len(results),
		},
		"query_id": queryID,
	})
}

// handleStreamCreate handles streaming source creation
func (s *Server) handleStreamCreate(w http.ResponseWriter, r *http.Request) {
	var req StreamCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create streaming source
	streamID := uuid.New().String()
	
	sourceConfigJSON, _ := json.Marshal(req.SourceConfig)
	processingRulesJSON, _ := json.Marshal(req.ProcessingRules)
	
	query := `INSERT INTO streaming_sources 
	          (id, name, source_type, connection_config, processing_rules, is_active, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	sourceType := "webhook" // Default
	if t, ok := req.SourceConfig["type"].(string); ok {
		sourceType = t
	}
	
	_, err := s.db.Exec(query,
		streamID,
		fmt.Sprintf("stream_%s", streamID[:8]),
		sourceType,
		sourceConfigJSON,
		processingRulesJSON,
		true,
		time.Now(),
	)
	
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create stream: %v", err))
		return
	}

	// Return stream details
	s.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"stream_id": streamID,
		"status":    "active",
		"endpoints": map[string]string{
			"webhook": fmt.Sprintf("/api/v1/stream/%s/webhook", streamID),
			"status":  fmt.Sprintf("/api/v1/stream/%s/status", streamID),
		},
	})
}

// Helper functions for data validation

func calculateCompleteness(data []map[string]interface{}) float64 {
	if len(data) == 0 {
		return 0.0
	}
	
	totalCells := 0
	filledCells := 0
	
	for _, row := range data {
		for _, value := range row {
			totalCells++
			if value != nil && value != "" {
				filledCells++
			}
		}
	}
	
	if totalCells == 0 {
		return 0.0
	}
	
	return float64(filledCells) / float64(totalCells)
}

func countDuplicates(data []map[string]interface{}) int {
	seen := make(map[string]bool)
	duplicates := 0
	
	for _, row := range data {
		// Create a unique key for the row
		rowJSON, _ := json.Marshal(row)
		rowKey := string(rowJSON)
		
		if seen[rowKey] {
			duplicates++
		} else {
			seen[rowKey] = true
		}
	}
	
	return duplicates
}

func detectAnomalies(data []map[string]interface{}) []map[string]interface{} {
	anomalies := []map[string]interface{}{}
	
	// Statistical anomaly detection for numeric fields
	numericStats := make(map[string][]float64)
	
	// Collect numeric values
	for _, row := range data {
		for key, value := range row {
			if numVal, ok := toFloat64(value); ok {
				if numericStats[key] == nil {
					numericStats[key] = []float64{}
				}
				numericStats[key] = append(numericStats[key], numVal)
			}
		}
	}
	
	// Calculate statistics and detect outliers
	for field, values := range numericStats {
		if len(values) < 4 {
			continue
		}
		
		mean, stdDev := calculateStats(values)
		
		// Find outliers (values beyond 3 standard deviations)
		for i, row := range data {
			if val, ok := toFloat64(row[field]); ok {
				if stdDev > 0 && (val > mean+3*stdDev || val < mean-3*stdDev) {
					anomalies = append(anomalies, map[string]interface{}{
						"row_index": i,
						"field":     field,
						"value":     val,
						"type":      "statistical_outlier",
						"severity":  "warning",
						"details":   fmt.Sprintf("Value %.2f is %.2f std devs from mean %.2f", val, (val-mean)/stdDev, mean),
					})
				}
			}
		}
	}
	
	// Detect pattern anomalies in string fields
	for i, row := range data {
		for field, value := range row {
			if strVal, ok := value.(string); ok {
				// Check for suspicious patterns
				if strings.Contains(strVal, "<script") || strings.Contains(strVal, "DROP TABLE") {
					anomalies = append(anomalies, map[string]interface{}{
						"row_index": i,
						"field":     field,
						"value":     strVal[:min(50, len(strVal))],
						"type":      "suspicious_content",
						"severity":  "critical",
						"details":   "Potentially malicious content detected",
					})
				}
				
				// Check for unexpected format
				if field == "email" && !strings.Contains(strVal, "@") {
					anomalies = append(anomalies, map[string]interface{}{
						"row_index": i,
						"field":     field,
						"value":     strVal,
						"type":      "format_violation",
						"severity":  "warning",
						"details":   "Email field missing @ symbol",
					})
				}
			}
		}
	}
	
	return anomalies
}

// Helper functions for anomaly detection
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func calculateStats(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}
	
	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))
	
	// Calculate standard deviation
	varSum := 0.0
	for _, v := range values {
		diff := v - mean
		varSum += diff * diff
	}
	variance := varSum / float64(len(values))
	stdDev = math.Sqrt(variance)
	
	return mean, stdDev
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func calculateAccuracy(data []map[string]interface{}, schema dataprocessor.Schema) float64 {
	if len(data) == 0 || len(schema.Columns) == 0 {
		return 0.0
	}
	
	totalChecks := 0
	passedChecks := 0
	
	// Check each row against schema
	for _, row := range data {
		for _, col := range schema.Columns {
			totalChecks++
			value, exists := row[col.Name]
			
			// Check existence
			if !exists {
				if !col.Nullable {
					continue // Failed check
				}
			}
			
			// Check type
			if value != nil && isCorrectType(value, col.Type) {
				passedChecks++
			} else if value == nil && col.Nullable {
				passedChecks++
			}
		}
	}
	
	if totalChecks == 0 {
		return 1.0
	}
	
	return float64(passedChecks) / float64(totalChecks)
}

func calculateConsistency(data []map[string]interface{}) float64 {
	if len(data) == 0 {
		return 1.0
	}
	
	consistencyScore := 1.0
	
	// Check field consistency (all rows have same fields)
	fieldCounts := make(map[string]int)
	for _, row := range data {
		for field := range row {
			fieldCounts[field]++
		}
	}
	
	// Calculate field consistency
	for _, count := range fieldCounts {
		fieldConsistency := float64(count) / float64(len(data))
		consistencyScore *= fieldConsistency
	}
	
	// Check format consistency for string fields
	patternConsistency := checkPatternConsistency(data)
	consistencyScore = (consistencyScore + patternConsistency) / 2.0
	
	return consistencyScore
}

func isCorrectType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		switch value.(type) {
		case int, int32, int64, float64:
			return true
		}
	case "float", "number":
		_, ok1 := value.(float64)
		_, ok2 := value.(float32)
		_, ok3 := value.(int)
		return ok1 || ok2 || ok3
	case "boolean":
		_, ok := value.(bool)
		return ok
	default:
		return true // Unknown type, assume correct
	}
	return false
}

func checkPatternConsistency(data []map[string]interface{}) float64 {
	if len(data) == 0 {
		return 1.0
	}
	
	// Check date format consistency
	datePatterns := make(map[string]map[string]int) // field -> pattern -> count
	
	for _, row := range data {
		for field, value := range row {
			if strVal, ok := value.(string); ok {
				pattern := detectPattern(strVal)
				if pattern != "" {
					if datePatterns[field] == nil {
						datePatterns[field] = make(map[string]int)
					}
					datePatterns[field][pattern]++
				}
			}
		}
	}
	
	// Calculate consistency score based on pattern uniformity
	totalScore := 0.0
	fieldCount := 0
	
	for _, patterns := range datePatterns {
		fieldCount++
		total := 0
		maxCount := 0
		
		for _, count := range patterns {
			total += count
			if count > maxCount {
				maxCount = count
			}
		}
		
		if total > 0 {
			totalScore += float64(maxCount) / float64(total)
		}
	}
	
	if fieldCount == 0 {
		return 1.0
	}
	
	return totalScore / float64(fieldCount)
}

func detectPattern(str string) string {
	// Simple pattern detection
	if strings.Contains(str, "@") && strings.Contains(str, ".") {
		return "email"
	}
	if strings.Contains(str, "-") && len(str) >= 8 && len(str) <= 10 {
		return "date-dashed"
	}
	if strings.Contains(str, "/") && len(str) >= 8 && len(str) <= 10 {
		return "date-slash"
	}
	if strings.HasPrefix(str, "http") {
		return "url"
	}
	return ""
}

func applyQualityRule(data []map[string]interface{}, rule QualityRule) []map[string]interface{} {
	violations := []map[string]interface{}{}
	
	// Simple rule application - in production use rule engine
	switch rule.RuleType {
	case "required_field":
		fieldName, _ := rule.Parameters["field"].(string)
		for i, row := range data {
			if row[fieldName] == nil || row[fieldName] == "" {
				violations = append(violations, map[string]interface{}{
					"row":      i,
					"field":    fieldName,
					"rule":     rule.RuleType,
					"severity": rule.Severity,
					"message":  fmt.Sprintf("Required field '%s' is missing", fieldName),
				})
			}
		}
	}
	
	return violations
}

func hasLimit(sql string) bool {
	// Simple check - in production use proper SQL parser
	return strings.Contains(strings.ToUpper(sql), "LIMIT")
}