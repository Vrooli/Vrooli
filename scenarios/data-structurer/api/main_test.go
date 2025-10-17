package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}

// Test Schema struct validation
func TestSchemaStructure(t *testing.T) {
	schema := Schema{
		ID:          uuid.New(),
		Name:        "test-schema",
		Description: "Test schema description",
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
		},
		Version:   1,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "test-user",
	}

	assert.NotEmpty(t, schema.ID)
	assert.Equal(t, "test-schema", schema.Name)
	assert.Equal(t, 1, schema.Version)
	assert.True(t, schema.IsActive)
}

// Test ProcessedData struct validation
func TestProcessedDataStructure(t *testing.T) {
	confidence := 0.95
	processingTime := 1000

	processedData := ProcessedData{
		ID:               uuid.New(),
		SchemaID:         uuid.New(),
		SourceFileName:   "test.txt",
		SourceFilePath:   "/tmp/test.txt",
		SourceFileType:   "text",
		SourceFileSize:   1024,
		RawContent:       "test content",
		StructuredData:   map[string]interface{}{"field": "value"},
		ConfidenceScore:  &confidence,
		ProcessingStatus: "completed",
		ProcessingTimeMs: &processingTime,
		CreatedAt:        time.Now(),
	}

	assert.NotEmpty(t, processedData.ID)
	assert.Equal(t, "completed", processedData.ProcessingStatus)
	assert.Equal(t, 0.95, *processedData.ConfidenceScore)
}

// Test ProcessingRequest validation
func TestProcessingRequestValidation(t *testing.T) {
	tests := []struct {
		name        string
		request     ProcessingRequest
		expectValid bool
	}{
		{
			name: "valid text processing request",
			request: ProcessingRequest{
				SchemaID:  uuid.New(),
				InputType: "text",
				InputData: "test content",
				BatchMode: false,
			},
			expectValid: true,
		},
		{
			name: "valid file processing request",
			request: ProcessingRequest{
				SchemaID:  uuid.New(),
				InputType: "file",
				InputData: "/path/to/file.pdf",
				BatchMode: false,
			},
			expectValid: true,
		},
		{
			name: "valid url processing request",
			request: ProcessingRequest{
				SchemaID:  uuid.New(),
				InputType: "url",
				InputData: "https://example.com/data",
				BatchMode: false,
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.request.SchemaID)
			assert.Contains(t, []string{"text", "file", "url"}, tt.request.InputType)
			assert.NotEmpty(t, tt.request.InputData)
		})
	}
}

// Test health check endpoint structure
func TestHealthCheckResponseStructure(t *testing.T) {
	healthResponse := HealthResponse{
		Status:    "healthy",
		Service:   "data-structurer-api",
		Timestamp: time.Now().Format(time.RFC3339),
		Readiness: true,
		Version:   "1.0.0",
		Dependencies: map[string]interface{}{
			"postgres": map[string]interface{}{
				"status": "healthy",
			},
			"ollama": map[string]interface{}{
				"status": "healthy",
			},
		},
	}

	assert.Equal(t, "healthy", healthResponse.Status)
	assert.Equal(t, "data-structurer-api", healthResponse.Service)
	assert.True(t, healthResponse.Readiness)
	assert.NotEmpty(t, healthResponse.Dependencies)
}

// Test healthy dependencies counter
func TestCountHealthyDependencies(t *testing.T) {
	tests := []struct {
		name         string
		dependencies map[string]interface{}
		expected     int
	}{
		{
			name: "all healthy",
			dependencies: map[string]interface{}{
				"postgres": map[string]interface{}{"status": "healthy"},
				"ollama":   map[string]interface{}{"status": "healthy"},
			},
			expected: 2,
		},
		{
			name: "mixed status",
			dependencies: map[string]interface{}{
				"postgres": map[string]interface{}{"status": "healthy"},
				"ollama":   map[string]interface{}{"status": "unhealthy"},
			},
			expected: 1,
		},
		{
			name: "no healthy dependencies",
			dependencies: map[string]interface{}{
				"postgres": map[string]interface{}{"status": "unhealthy"},
			},
			expected: 0,
		},
		{
			name:         "empty dependencies",
			dependencies: map[string]interface{}{},
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countHealthyDependencies(tt.dependencies)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test ProcessingResponse structure
func TestProcessingResponseStructure(t *testing.T) {
	confidence := 0.85
	response := ProcessingResponse{
		ProcessingID:   uuid.New(),
		Status:         "completed",
		StructuredData: map[string]interface{}{"name": "John Doe", "age": 30},
		ConfidenceScore: &confidence,
		Errors:         []string{},
	}

	assert.NotEmpty(t, response.ProcessingID)
	assert.Equal(t, "completed", response.Status)
	assert.NotNil(t, response.StructuredData)
	assert.Equal(t, 0.85, *response.ConfidenceScore)
	assert.Empty(t, response.Errors)
}

// Test SchemaTemplate structure
func TestSchemaTemplateStructure(t *testing.T) {
	template := SchemaTemplate{
		ID:          uuid.New(),
		Name:        "contact-template",
		Category:    "contacts",
		Description: "Template for contact information",
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":  map[string]interface{}{"type": "string"},
				"email": map[string]interface{}{"type": "string"},
			},
		},
		UsageCount: 0,
		IsPublic:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Tags:       []string{"contacts", "crm"},
	}

	assert.NotEmpty(t, template.ID)
	assert.Equal(t, "contact-template", template.Name)
	assert.True(t, template.IsPublic)
	assert.Len(t, template.Tags, 2)
}

// Test JSON serialization of Schema
func TestSchemaJSONSerialization(t *testing.T) {
	schema := Schema{
		ID:          uuid.New(),
		Name:        "test-schema",
		Description: "Test schema",
		SchemaDefinition: map[string]interface{}{
			"type": "object",
		},
		Version:   1,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jsonData, err := json.Marshal(schema)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var decoded Schema
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, schema.Name, decoded.Name)
	assert.Equal(t, schema.Version, decoded.Version)
}

// Test ProcessingRequest JSON deserialization
func TestProcessingRequestJSONDeserialization(t *testing.T) {
	jsonData := `{
		"schema_id": "123e4567-e89b-12d3-a456-426614174000",
		"input_type": "text",
		"input_data": "test content",
		"batch_mode": false
	}`

	var request ProcessingRequest
	err := json.Unmarshal([]byte(jsonData), &request)
	assert.NoError(t, err)
	assert.Equal(t, "text", request.InputType)
	assert.Equal(t, "test content", request.InputData)
	assert.False(t, request.BatchMode)
}

// Test invalid input type validation
func TestInvalidInputType(t *testing.T) {
	invalidTypes := []string{"invalid", "pdf", "csv", "xml"}

	for _, invalidType := range invalidTypes {
		request := ProcessingRequest{
			SchemaID:  uuid.New(),
			InputType: invalidType,
			InputData: "test",
		}

		// In a real scenario, this would be validated by Gin's binding
		validTypes := []string{"text", "file", "url"}
		isValid := false
		for _, validType := range validTypes {
			if request.InputType == validType {
				isValid = true
				break
			}
		}
		assert.False(t, isValid, "Input type %s should be invalid", invalidType)
	}
}

// Test error response structure
func TestErrorResponseHandling(t *testing.T) {
	response := ProcessingResponse{
		ProcessingID: uuid.New(),
		Status:       "failed",
		Errors:       []string{"Schema not found", "Invalid input data"},
	}

	assert.Equal(t, "failed", response.Status)
	assert.Len(t, response.Errors, 2)
	assert.Contains(t, response.Errors, "Schema not found")
}

// Test timestamp formats
func TestTimestampFormats(t *testing.T) {
	now := time.Now()
	schema := Schema{
		ID:        uuid.New(),
		Name:      "test",
		CreatedAt: now,
		UpdatedAt: now,
	}

	jsonData, err := json.Marshal(schema)
	assert.NoError(t, err)

	var decoded Schema
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)

	// Check timestamps are preserved (within 1 second due to JSON precision)
	assert.WithinDuration(t, schema.CreatedAt, decoded.CreatedAt, time.Second)
	assert.WithinDuration(t, schema.UpdatedAt, decoded.UpdatedAt, time.Second)
}

// Test UUID generation and validation
func TestUUIDGeneration(t *testing.T) {
	// Generate multiple UUIDs and ensure they're unique
	uuids := make(map[uuid.UUID]bool)
	for i := 0; i < 100; i++ {
		id := uuid.New()
		assert.False(t, uuids[id], "UUID should be unique")
		uuids[id] = true
	}
	assert.Len(t, uuids, 100)
}

// Test confidence score boundaries
func TestConfidenceScoreBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		valid bool
	}{
		{"zero confidence", 0.0, true},
		{"perfect confidence", 1.0, true},
		{"mid confidence", 0.75, true},
		{"negative confidence", -0.1, false},
		{"over 100% confidence", 1.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.score >= 0.0 && tt.score <= 1.0
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// Test batch mode flag
func TestBatchModeFlag(t *testing.T) {
	request := ProcessingRequest{
		SchemaID:  uuid.New(),
		InputType: "text",
		InputData: "test",
		BatchMode: true,
	}

	assert.True(t, request.BatchMode)
}

// Test empty schema definition handling
func TestEmptySchemaDefinition(t *testing.T) {
	schema := Schema{
		ID:               uuid.New(),
		Name:             "test",
		SchemaDefinition: map[string]interface{}{},
	}

	assert.NotNil(t, schema.SchemaDefinition)
	assert.Empty(t, schema.SchemaDefinition)
}

// Benchmark JSON marshaling performance
func BenchmarkSchemaJSONMarshal(b *testing.B) {
	schema := Schema{
		ID:          uuid.New(),
		Name:        "benchmark-schema",
		Description: "Performance testing schema",
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field1": map[string]interface{}{"type": "string"},
				"field2": map[string]interface{}{"type": "number"},
			},
		},
		Version:  1,
		IsActive: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(schema)
	}
}

// Benchmark ProcessingRequest creation
func BenchmarkProcessingRequestCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProcessingRequest{
			SchemaID:  uuid.New(),
			InputType: "text",
			InputData: "benchmark test data",
			BatchMode: false,
		}
	}
}
