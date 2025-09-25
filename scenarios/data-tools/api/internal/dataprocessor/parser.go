package dataprocessor

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DataFormat represents the format of input data
type DataFormat string

const (
	FormatCSV   DataFormat = "csv"
	FormatJSON  DataFormat = "json"
	FormatXML   DataFormat = "xml"
	FormatExcel DataFormat = "excel"
	FormatAuto  DataFormat = "auto"
)

// Column represents a data column schema
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// Schema represents the structure of a dataset
type Schema struct {
	Columns      []Column `json:"columns"`
	RowCount     int      `json:"row_count"`
	QualityScore float64  `json:"quality_score"`
}

// ParseResult contains the result of data parsing
type ParseResult struct {
	Schema   Schema        `json:"schema"`
	Preview  []interface{} `json:"preview"`
	Warnings []string      `json:"warnings"`
}

// ParseOptions contains options for data parsing
type ParseOptions struct {
	Delimiter   string `json:"delimiter"`
	Headers     bool   `json:"headers"`
	InferTypes  bool   `json:"infer_types"`
	SampleSize  int    `json:"sample_size"`
}

// Parser handles data parsing operations
type Parser struct{}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses data and infers schema
func (p *Parser) Parse(data string, format DataFormat, options ParseOptions) (*ParseResult, error) {
	if format == FormatAuto {
		format = p.detectFormat(data)
	}

	switch format {
	case FormatCSV:
		return p.parseCSV(data, options)
	case FormatJSON:
		return p.parseJSON(data, options)
	case FormatXML:
		return p.parseXML(data, options)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// detectFormat attempts to detect the data format
func (p *Parser) detectFormat(data string) DataFormat {
	trimmed := strings.TrimSpace(data)
	
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return FormatJSON
	}
	if strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<") {
		return FormatXML
	}
	
	return FormatCSV
}

// parseCSV parses CSV data
func (p *Parser) parseCSV(data string, options ParseOptions) (*ParseResult, error) {
	reader := csv.NewReader(strings.NewReader(data))
	
	if options.Delimiter != "" {
		reader.Comma = rune(options.Delimiter[0])
	}
	
	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	
	if len(records) == 0 {
		return &ParseResult{
			Schema:   Schema{Columns: []Column{}, RowCount: 0},
			Preview:  []interface{}{},
			Warnings: []string{"No data found"},
		}, nil
	}
	
	var headers []string
	var dataRows [][]string
	
	if options.Headers && len(records) > 0 {
		headers = records[0]
		if len(records) > 1 {
			dataRows = records[1:]
		}
	} else {
		// Generate column names
		if len(records) > 0 {
			for i := range records[0] {
				headers = append(headers, fmt.Sprintf("column_%d", i+1))
			}
		}
		dataRows = records
	}
	
	// Infer column types
	columns := make([]Column, len(headers))
	for i, header := range headers {
		columns[i] = Column{
			Name:     header,
			Type:     "string",
			Nullable: false,
		}
		
		if options.InferTypes && len(dataRows) > 0 {
			columns[i].Type = p.inferColumnType(dataRows, i)
		}
	}
	
	// Create preview
	preview := []interface{}{}
	sampleSize := options.SampleSize
	if sampleSize == 0 || sampleSize > len(dataRows) {
		sampleSize = len(dataRows)
	}
	if sampleSize > 10 {
		sampleSize = 10
	}
	
	for i := 0; i < sampleSize && i < len(dataRows); i++ {
		row := make(map[string]interface{})
		for j, value := range dataRows[i] {
			if j < len(headers) {
				row[headers[j]] = p.convertValue(value, columns[j].Type)
			}
		}
		preview = append(preview, row)
	}
	
	// Calculate quality score
	qualityScore := p.calculateQualityScore(dataRows, columns)
	
	return &ParseResult{
		Schema: Schema{
			Columns:      columns,
			RowCount:     len(dataRows),
			QualityScore: qualityScore,
		},
		Preview:  preview,
		Warnings: []string{},
	}, nil
}

// parseJSON parses JSON data
func (p *Parser) parseJSON(data string, options ParseOptions) (*ParseResult, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Handle array of objects
	if arr, ok := jsonData.([]interface{}); ok {
		return p.parseJSONArray(arr, options)
	}
	
	// Handle single object
	if obj, ok := jsonData.(map[string]interface{}); ok {
		return p.parseJSONArray([]interface{}{obj}, options)
	}
	
	return nil, fmt.Errorf("unsupported JSON structure")
}

// parseJSONArray parses an array of JSON objects
func (p *Parser) parseJSONArray(arr []interface{}, options ParseOptions) (*ParseResult, error) {
	if len(arr) == 0 {
		return &ParseResult{
			Schema:   Schema{Columns: []Column{}, RowCount: 0},
			Preview:  []interface{}{},
			Warnings: []string{"Empty array"},
		}, nil
	}
	
	// Collect all unique keys
	keyMap := make(map[string]string)
	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			for key, value := range obj {
				if _, exists := keyMap[key]; !exists {
					keyMap[key] = p.inferJSONType(value)
				}
			}
		}
	}
	
	// Create columns
	columns := []Column{}
	for key, dataType := range keyMap {
		columns = append(columns, Column{
			Name:     key,
			Type:     dataType,
			Nullable: false,
		})
	}
	
	// Create preview
	sampleSize := options.SampleSize
	if sampleSize == 0 || sampleSize > len(arr) {
		sampleSize = len(arr)
	}
	if sampleSize > 10 {
		sampleSize = 10
	}
	
	preview := arr[:sampleSize]
	
	return &ParseResult{
		Schema: Schema{
			Columns:      columns,
			RowCount:     len(arr),
			QualityScore: 0.95, // JSON typically has good quality
		},
		Preview:  preview,
		Warnings: []string{},
	}, nil
}

// parseXML parses XML data
func (p *Parser) parseXML(data string, options ParseOptions) (*ParseResult, error) {
	// Simple XML parsing - in production, use more sophisticated XML parsing
	decoder := xml.NewDecoder(strings.NewReader(data))
	
	elements := []map[string]interface{}{}
	current := make(map[string]interface{})
	
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}
		
		switch se := token.(type) {
		case xml.StartElement:
			current[se.Name.Local] = ""
		case xml.CharData:
			// Store character data
			for key := range current {
				current[key] = string(se)
			}
		case xml.EndElement:
			if len(current) > 0 {
				elements = append(elements, current)
				current = make(map[string]interface{})
			}
		}
	}
	
	if len(elements) == 0 {
		return &ParseResult{
			Schema:   Schema{Columns: []Column{}, RowCount: 0},
			Preview:  []interface{}{},
			Warnings: []string{"No data elements found in XML"},
		}, nil
	}
	
	// Infer schema from elements
	keyMap := make(map[string]bool)
	for _, elem := range elements {
		for key := range elem {
			keyMap[key] = true
		}
	}
	
	columns := []Column{}
	for key := range keyMap {
		columns = append(columns, Column{
			Name:     key,
			Type:     "string",
			Nullable: false,
		})
	}
	
	// Create preview
	sampleSize := options.SampleSize
	if sampleSize == 0 || sampleSize > len(elements) {
		sampleSize = len(elements)
	}
	if sampleSize > 10 {
		sampleSize = 10
	}
	
	preview := []interface{}{}
	for i := 0; i < sampleSize; i++ {
		preview = append(preview, elements[i])
	}
	
	return &ParseResult{
		Schema: Schema{
			Columns:      columns,
			RowCount:     len(elements),
			QualityScore: 0.85,
		},
		Preview:  preview,
		Warnings: []string{},
	}, nil
}

// inferColumnType infers the data type of a column
func (p *Parser) inferColumnType(rows [][]string, colIndex int) string {
	intCount := 0
	floatCount := 0
	boolCount := 0
	
	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		
		value := strings.TrimSpace(row[colIndex])
		if value == "" {
			continue
		}
		
		// Check for boolean
		if value == "true" || value == "false" {
			boolCount++
			continue
		}
		
		// Check for integer
		if _, err := strconv.ParseInt(value, 10, 64); err == nil {
			intCount++
			continue
		}
		
		// Check for float
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			floatCount++
			continue
		}
	}
	
	total := len(rows)
	if boolCount > total/2 {
		return "boolean"
	}
	if intCount > total/2 {
		return "integer"
	}
	if floatCount > total/2 {
		return "float"
	}
	
	return "string"
}

// inferJSONType infers the type of a JSON value
func (p *Parser) inferJSONType(value interface{}) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case float64:
		// Check if it's actually an integer
		if v, ok := value.(float64); ok && v == float64(int64(v)) {
			return "integer"
		}
		return "float"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// convertValue converts a string value to the appropriate type
func (p *Parser) convertValue(value string, dataType string) interface{} {
	switch dataType {
	case "integer":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return v
		}
	case "float":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
	case "boolean":
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return value
}

// calculateQualityScore calculates a quality score for the data
func (p *Parser) calculateQualityScore(rows [][]string, columns []Column) float64 {
	if len(rows) == 0 {
		return 0.0
	}
	
	totalCells := len(rows) * len(columns)
	validCells := 0
	
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(columns) {
				break
			}
			
			if cell != "" {
				validCells++
			}
		}
	}
	
	completeness := float64(validCells) / float64(totalCells)
	
	// Basic quality score based on completeness
	// In production, add more sophisticated quality metrics
	return completeness
}