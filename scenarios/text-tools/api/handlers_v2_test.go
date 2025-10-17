package main

import (
	"net/http"
	"testing"
)

func TestDiffHandlerV2(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_WithRequestID", func(t *testing.T) {
		reqBody := DiffRequestV2{
			DiffRequest: DiffRequest{
				Text1: "hello",
				Text2: "world",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateV2Response(t, response)
			Validator.ValidateRequiredFields(t, response, []string{"changes", "similarity_score", "request_id"})
		}

		// Check v2 header
		apiVersion := w.Header().Get("X-API-Version")
		if apiVersion != "2.0" {
			t.Errorf("Expected X-API-Version 2.0, got %s", apiVersion)
		}
	})

	t.Run("Success_WithPatch", func(t *testing.T) {
		reqBody := DiffRequestV2{
			DiffRequest: DiffRequest{
				Text1: "hello",
				Text2: "world",
			},
			Options: DiffOptionsV2{
				GeneratePatch: true,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if _, exists := response["patch"]; !exists {
				t.Error("Expected patch field in response when GeneratePatch is true")
			}
		}
	})

	t.Run("Error_Validation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"text1": "hello",
			// Missing text2
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/diff",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.DiffHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestSearchHandlerV2(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_WithRequestID", func(t *testing.T) {
		reqBody := SearchRequestV2{
			SearchRequest: SearchRequest{
				Text:    "hello world",
				Pattern: "hello",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/search",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.SearchHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateV2Response(t, response)
		}
	})

	t.Run("Error_NegativeMaxResults", func(t *testing.T) {
		reqBody := SearchRequestV2{
			SearchRequest: SearchRequest{
				Text:    "hello world",
				Pattern: "hello",
			},
			Options: SearchOptionsV2{
				MaxResults: -1,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/search",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.SearchHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for negative max_results, got %d", w.Code)
		}
	})
}

func TestTransformHandlerV2(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_WithIntermediates", func(t *testing.T) {
		reqBody := TransformRequestV2{
			TransformRequest: TransformRequest{
				Text: "hello",
				Transformations: []Transformation{
					{Type: "upper"},
					{Type: "lower"},
				},
			},
			Options: TransformOptionsV2{
				TrackIntermediates: true,
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/transform",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.TransformHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if intermediates, exists := response["intermediate_results"]; exists {
				if intermediatesList, ok := intermediates.([]interface{}); ok {
					if len(intermediatesList) == 0 {
						t.Error("Expected intermediate results when TrackIntermediates is true")
					}
				}
			}
		}
	})

	t.Run("Error_EmptyTransformations", func(t *testing.T) {
		reqBody := TransformRequestV2{
			TransformRequest: TransformRequest{
				Text:            "hello",
				Transformations: []Transformation{},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/transform",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.TransformHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty transformations, got %d", w.Code)
		}
	})
}

func TestExtractHandlerV2(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_WithStructuredData", func(t *testing.T) {
		reqBody := ExtractRequestV2{
			ExtractRequest: ExtractRequest{
				Source: map[string]interface{}{
					"text": "Test content",
				},
			},
			Options: ExtractOptionsV2{
				ExtractOptions: ExtractOptions{
					ExtractStructured: true,
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateV2Response(t, response)
		}
	})

	t.Run("Error_MissingSource", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"format": "text",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/extract",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.ExtractHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing source, got %d", w.Code)
		}
	})
}

func TestAnalyzeHandlerV2(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_WithAI", func(t *testing.T) {
		reqBody := AnalyzeRequestV2{
			AnalyzeRequest: AnalyzeRequest{
				Text:     "This is a test text for analysis",
				Analyses: []string{"statistics"},
				Options: AnalyzeOptions{
					UseAI: true,
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/analyze",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.AnalyzeHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateV2Response(t, response)
		}
	})

	t.Run("Error_EmptyAnalyses", func(t *testing.T) {
		reqBody := AnalyzeRequestV2{
			AnalyzeRequest: AnalyzeRequest{
				Text:     "test",
				Analyses: []string{},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/analyze",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.AnalyzeHandlerV2)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty analyses, got %d", w.Code)
		}
	})
}

func TestPipelineHandler(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Success_MultiplSteps", func(t *testing.T) {
		reqBody := PipelineRequest{
			Input: "hello world",
			Steps: []PipelineStep{
				{
					Name:      "uppercase",
					Operation: "transform",
					Parameters: map[string]interface{}{
						"type": "upper",
					},
				},
				{
					Name:      "lowercase",
					Operation: "transform",
					Parameters: map[string]interface{}{
						"type": "lower",
					},
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/pipeline",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.PipelineHandler)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			Validator.ValidateRequiredFields(t, response, []string{"final_output", "steps", "request_id"})

			if steps, ok := response["steps"].([]interface{}); ok {
				if len(steps) != 2 {
					t.Errorf("Expected 2 steps in response, got %d", len(steps))
				}
			}
		}
	})

	t.Run("Error_EmptyInput", func(t *testing.T) {
		reqBody := PipelineRequest{
			Input: "",
			Steps: []PipelineStep{
				{Name: "test", Operation: "transform"},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/pipeline",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.PipelineHandler)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty input, got %d", w.Code)
		}
	})

	t.Run("Error_EmptySteps", func(t *testing.T) {
		reqBody := PipelineRequest{
			Input: "test",
			Steps: []PipelineStep{},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/pipeline",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.PipelineHandler)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty steps, got %d", w.Code)
		}
	})

	t.Run("Error_StepMissingName", func(t *testing.T) {
		reqBody := PipelineRequest{
			Input: "test",
			Steps: []PipelineStep{
				{
					Name:      "",
					Operation: "transform",
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v2/pipeline",
			Body:   reqBody,
		}

		w, err := makeHTTPRequest(req, env.Server.PipelineHandler)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for step missing name, got %d", w.Code)
		}
	})
}

func TestValidationMethods(t *testing.T) {
	t.Run("DiffRequestV2_Validate", func(t *testing.T) {
		validReq := &DiffRequestV2{
			DiffRequest: DiffRequest{
				Text1: "hello",
				Text2: "world",
			},
		}

		if err := validReq.Validate(); err != nil {
			t.Errorf("Expected no error for valid request, got %v", err)
		}

		invalidReq := &DiffRequestV2{
			DiffRequest: DiffRequest{
				Text1: "hello",
			},
		}

		if err := invalidReq.Validate(); err == nil {
			t.Error("Expected error for missing text2")
		}
	})

	t.Run("SearchRequestV2_Validate", func(t *testing.T) {
		validReq := &SearchRequestV2{
			SearchRequest: SearchRequest{
				Pattern: "test",
			},
		}

		if err := validReq.Validate(); err != nil {
			t.Errorf("Expected no error for valid request, got %v", err)
		}

		invalidReq := &SearchRequestV2{
			SearchRequest: SearchRequest{
				Pattern: "",
			},
		}

		if err := invalidReq.Validate(); err == nil {
			t.Error("Expected error for empty pattern")
		}

		negativeMaxReq := &SearchRequestV2{
			SearchRequest: SearchRequest{
				Pattern: "test",
			},
			Options: SearchOptionsV2{
				MaxResults: -1,
			},
		}

		if err := negativeMaxReq.Validate(); err == nil {
			t.Error("Expected error for negative max_results")
		}
	})

	t.Run("TransformRequestV2_Validate", func(t *testing.T) {
		validReq := &TransformRequestV2{
			TransformRequest: TransformRequest{
				Text: "hello",
				Transformations: []Transformation{
					{Type: "upper"},
				},
			},
		}

		if err := validReq.Validate(); err != nil {
			t.Errorf("Expected no error for valid request, got %v", err)
		}

		emptyTextReq := &TransformRequestV2{
			TransformRequest: TransformRequest{
				Text: "",
				Transformations: []Transformation{
					{Type: "upper"},
				},
			},
		}

		if err := emptyTextReq.Validate(); err == nil {
			t.Error("Expected error for empty text")
		}

		emptyTransformsReq := &TransformRequestV2{
			TransformRequest: TransformRequest{
				Text:            "hello",
				Transformations: []Transformation{},
			},
		}

		if err := emptyTransformsReq.Validate(); err == nil {
			t.Error("Expected error for empty transformations")
		}
	})

	t.Run("PipelineRequest_Validate", func(t *testing.T) {
		validReq := &PipelineRequest{
			Input: "test",
			Steps: []PipelineStep{
				{Name: "step1", Operation: "transform"},
			},
		}

		if err := validReq.Validate(); err != nil {
			t.Errorf("Expected no error for valid request, got %v", err)
		}

		emptyInputReq := &PipelineRequest{
			Input: "",
			Steps: []PipelineStep{
				{Name: "step1", Operation: "transform"},
			},
		}

		if err := emptyInputReq.Validate(); err == nil {
			t.Error("Expected error for empty input")
		}

		emptyStepsReq := &PipelineRequest{
			Input: "test",
			Steps: []PipelineStep{},
		}

		if err := emptyStepsReq.Validate(); err == nil {
			t.Error("Expected error for empty steps")
		}

		missingNameReq := &PipelineRequest{
			Input: "test",
			Steps: []PipelineStep{
				{Name: "", Operation: "transform"},
			},
		}

		if err := missingNameReq.Validate(); err == nil {
			t.Error("Expected error for step missing name")
		}
	})
}
