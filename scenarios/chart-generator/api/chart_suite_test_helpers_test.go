package main

import (
	"context"
	"testing"
)

type chartGenerationCase struct {
	name           string
	req            ChartGenerationProcessorRequest
	expectSuccess  bool
	expectedError  string
	validateResult func(t *testing.T, result *ChartGenerationProcessorResponse)
}

func runChartGenerationCases(t *testing.T, tests []chartGenerationCase) {
	t.Helper()
	cleanup := setupTestLogger()
	defer cleanup()
	processor := NewChartProcessor(nil)
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.GenerateChart(ctx, tt.req)
			if err != nil {
				t.Fatalf("GenerateChart returned unexpected error: %v", err)
			}
			if tt.expectSuccess {
				if !result.Success {
					errorMsg := ""
					if result.Error != nil {
						errorMsg = result.Error.Message
					}
					t.Errorf("Expected success, got failure: %s", errorMsg)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
				return
			}
			if result.Success {
				t.Error("Expected failure, got success")
			}
			if result.Error == nil {
				t.Error("Expected error in response, got nil")
			} else if tt.expectedError != "" && !containsString(result.Error.Message, tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, result.Error.Message)
			}
		})
	}
}

func assertThemeAvailable(t *testing.T, styleID string) {
	t.Helper()
	cleanup := setupTestLogger()
	defer cleanup()
	result, err := NewChartProcessor(nil).GetAvailableStyles(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableStyles returned error: %v", err)
	}
	for _, style := range result.Styles {
		if style.ID == styleID {
			return
		}
	}
	t.Errorf("Expected %s theme to be available", styleID)
}

type chartExportCase struct {
	name    string
	chartID string
	req     ChartGenerationProcessorRequest
}

func runChartExportCases(t *testing.T, tests []chartExportCase) {
	t.Helper()
	cleanup := setupTestLogger()
	defer cleanup()
	env := setupTestDirectory(t)
	defer env.Cleanup()
	renderer := NewChartRenderer(env.TempDir)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := renderer.RenderChart(context.Background(), tt.chartID, tt.req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(files) == 0 {
				t.Fatal("Expected at least one file to be generated")
			}
			for format, path := range files {
				assertFileExists(t, path)
				t.Logf("Generated %s file: %s", format, path)
			}
		})
	}
}
