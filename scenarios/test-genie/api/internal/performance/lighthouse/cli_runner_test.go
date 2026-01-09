package lighthouse

import (
	"context"
	"testing"
	"time"
)

func TestNewCLIRunner(t *testing.T) {
	runner := NewCLIRunner()
	if runner == nil {
		t.Fatal("NewCLIRunner returned nil")
	}
	if runner.timeout != 120*time.Second {
		t.Errorf("default timeout = %v, want %v", runner.timeout, 120*time.Second)
	}
}

func TestNewCLIRunner_WithOptions(t *testing.T) {
	runner := NewCLIRunner(
		WithCLITimeout(60*time.Second),
		WithChromePath("/usr/bin/chrome"),
	)
	if runner.timeout != 60*time.Second {
		t.Errorf("timeout = %v, want %v", runner.timeout, 60*time.Second)
	}
	if runner.chromePath != "/usr/bin/chrome" {
		t.Errorf("chromePath = %q, want %q", runner.chromePath, "/usr/bin/chrome")
	}
}

func TestCLIRunner_BuildArgs(t *testing.T) {
	runner := NewCLIRunner()

	tests := []struct {
		name     string
		req      AuditRequest
		contains []string
	}{
		{
			name: "basic request",
			req: AuditRequest{
				URL: "http://example.com",
			},
			contains: []string{
				"http://example.com",
				"--output=json",
				"--chrome-flags=--headless --no-sandbox --disable-gpu",
			},
		},
		{
			name: "with categories",
			req: AuditRequest{
				URL: "http://example.com",
				Config: map[string]interface{}{
					"settings": map[string]interface{}{
						"onlyCategories": []string{"performance", "accessibility"},
					},
				},
			},
			contains: []string{
				"--only-categories=performance,accessibility",
			},
		},
		{
			name: "with desktop formFactor",
			req: AuditRequest{
				URL: "http://example.com",
				Config: map[string]interface{}{
					"settings": map[string]interface{}{
						"formFactor": "desktop",
					},
				},
			},
			contains: []string{
				"--preset=desktop",
			},
		},
		{
			name: "with mobile formFactor",
			req: AuditRequest{
				URL: "http://example.com",
				Config: map[string]interface{}{
					"settings": map[string]interface{}{
						"formFactor": "mobile",
					},
				},
			},
			contains: []string{
				"--form-factor=mobile",
			},
		},
		{
			name: "with throttling method",
			req: AuditRequest{
				URL: "http://example.com",
				Config: map[string]interface{}{
					"settings": map[string]interface{}{
						"throttlingMethod": "simulate",
					},
				},
			},
			contains: []string{
				"--throttling-method=simulate",
			},
		},
		{
			name: "with HTML output",
			req: AuditRequest{
				URL:         "http://example.com",
				IncludeHTML: true,
			},
			contains: []string{
				"--output=json,html",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := runner.buildArgs(tt.req, "/tmp/report.json")

			for _, expected := range tt.contains {
				found := false
				for _, arg := range args {
					if arg == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("args missing %q, got %v", expected, args)
				}
			}
		})
	}
}

func TestCLIRunner_BuildArgs_CategoriesAsInterface(t *testing.T) {
	runner := NewCLIRunner()

	// Test with categories as []interface{} (as it comes from JSON unmarshaling)
	req := AuditRequest{
		URL: "http://example.com",
		Config: map[string]interface{}{
			"settings": map[string]interface{}{
				"onlyCategories": []interface{}{"performance", "seo"},
			},
		},
	}

	args := runner.buildArgs(req, "/tmp/report.json")

	found := false
	for _, arg := range args {
		if arg == "--only-categories=performance,seo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("args missing categories flag, got %v", args)
	}
}

func TestParseLighthouseOutput_Valid(t *testing.T) {
	input := []byte(`{
		"categories": {
			"performance": {
				"id": "performance",
				"title": "Performance",
				"score": 0.95
			},
			"accessibility": {
				"id": "accessibility",
				"title": "Accessibility",
				"score": 0.87
			}
		},
		"audits": {}
	}`)

	resp, err := parseLighthouseOutput(input)
	if err != nil {
		t.Fatalf("parseLighthouseOutput error: %v", err)
	}

	if len(resp.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(resp.Categories))
	}

	if perf, ok := resp.Categories["performance"]; !ok {
		t.Error("missing performance category")
	} else if perf.Score == nil || *perf.Score != 0.95 {
		t.Errorf("performance score = %v, want 0.95", perf.Score)
	}

	if acc, ok := resp.Categories["accessibility"]; !ok {
		t.Error("missing accessibility category")
	} else if acc.Score == nil || *acc.Score != 0.87 {
		t.Errorf("accessibility score = %v, want 0.87", acc.Score)
	}
}

func TestParseLighthouseOutput_NullScore(t *testing.T) {
	input := []byte(`{
		"categories": {
			"pwa": {
				"id": "pwa",
				"title": "PWA",
				"score": null
			}
		}
	}`)

	resp, err := parseLighthouseOutput(input)
	if err != nil {
		t.Fatalf("parseLighthouseOutput error: %v", err)
	}

	if pwa, ok := resp.Categories["pwa"]; !ok {
		t.Error("missing pwa category")
	} else if pwa.Score != nil {
		t.Errorf("pwa score = %v, want nil", pwa.Score)
	}
}

func TestParseLighthouseOutput_Empty(t *testing.T) {
	_, err := parseLighthouseOutput([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseLighthouseOutput_InvalidJSON(t *testing.T) {
	_, err := parseLighthouseOutput([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		got := truncateString(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestCLIRunner_Audit_EmptyURL(t *testing.T) {
	runner := NewCLIRunner()
	ctx := context.Background()

	_, err := runner.Audit(ctx, AuditRequest{URL: ""})
	if err == nil {
		t.Error("expected error for empty URL")
	}
}
