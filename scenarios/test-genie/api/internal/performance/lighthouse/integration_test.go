// Package lighthouse provides Lighthouse audit functionality.
//
// This file contains integration tests that require the Lighthouse CLI.
// These tests are skipped when Lighthouse is unavailable.
package lighthouse

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"
)

// checkLighthouseAvailable checks if Lighthouse CLI is available and skips the test if not.
func checkLighthouseAvailable(t *testing.T) {
	t.Helper()

	// Try to find lighthouse directly
	if _, err := exec.LookPath("lighthouse"); err == nil {
		return
	}

	// Try npx
	if _, err := exec.LookPath("npx"); err == nil {
		// Check if npx can run lighthouse
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "npx", "lighthouse", "--version")
		if err := cmd.Run(); err == nil {
			return
		}
	}

	t.Skip("Skipping integration test: Lighthouse CLI not available. Install with: npm install -g lighthouse")
}

// checkChromeAvailable does a basic check for Chrome availability.
// Note: Lighthouse itself will give a better error if Chrome is missing.
func checkChromeAvailable(t *testing.T) {
	t.Helper()

	// Common Chrome paths
	chromePaths := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	}

	for _, path := range chromePaths {
		if _, err := exec.LookPath(path); err == nil {
			return
		}
	}

	// Also check CHROME_PATH env var - if set, assume it's valid
	// The actual validation will happen when Lighthouse runs
	t.Log("Warning: Could not find Chrome in common paths. Lighthouse will attempt to locate it.")
}

// TestIntegration_CLIRunner_Health tests the CLI runner's health check.
func TestIntegration_CLIRunner_Health(t *testing.T) {
	checkLighthouseAvailable(t)

	runner := NewCLIRunner()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := runner.Health(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

// TestIntegration_CLIRunner_Audit tests the CLI runner's audit method.
func TestIntegration_CLIRunner_Audit(t *testing.T) {
	checkLighthouseAvailable(t)
	checkChromeAvailable(t)

	// Create a simple test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="Test page for Lighthouse integration tests">
    <title>Test Page</title>
    <style>
        body { font-family: sans-serif; margin: 20px; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <h1>Lighthouse Integration Test Page</h1>
    <p>This is a simple test page for Lighthouse audits.</p>
    <main>
        <article>
            <h2>Test Content</h2>
            <p>Content for testing accessibility and SEO.</p>
        </article>
    </main>
</body>
</html>`)
	}))
	defer testServer.Close()

	runner := NewCLIRunner(WithCLITimeout(90 * time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := AuditRequest{
		URL: testServer.URL,
		Config: map[string]interface{}{
			"settings": map[string]interface{}{
				"onlyCategories":   []string{"performance", "accessibility"},
				"throttlingMethod": "simulate",
				"formFactor":       "desktop",
			},
		},
	}

	resp, err := runner.Audit(ctx, req)
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}

	// Verify we got categories back
	if len(resp.Categories) == 0 {
		t.Fatal("Expected categories in response")
	}

	// Log what we got
	for catName, cat := range resp.Categories {
		score := "nil"
		if cat.Score != nil {
			score = fmt.Sprintf("%.0f%%", *cat.Score*100)
		}
		t.Logf("Category %s: %s", catName, score)
	}

	// Verify expected categories are present
	expectedCategories := []string{"performance", "accessibility"}
	for _, expected := range expectedCategories {
		if _, ok := resp.Categories[expected]; !ok {
			t.Errorf("Expected category %q not found in response", expected)
		}
	}
}

// TestIntegration_Validator_RealLighthouseAudit tests the full validator with real Lighthouse.
func TestIntegration_Validator_RealLighthouseAudit(t *testing.T) {
	checkLighthouseAvailable(t)
	checkChromeAvailable(t)

	// Create a simple test HTTP server with a basic HTML page
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="Test page for Lighthouse integration tests">
    <title>Test Page</title>
    <style>
        body { font-family: sans-serif; margin: 20px; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <h1>Lighthouse Integration Test Page</h1>
    <p>This is a simple test page for Lighthouse audits.</p>
    <main>
        <article>
            <h2>Test Content</h2>
            <p>Content for testing accessibility and SEO.</p>
        </article>
    </main>
</body>
</html>`)
	}))
	defer testServer.Close()

	t.Run("Validator_AuditSucceeds", func(t *testing.T) {
		config := &Config{
			Enabled: true,
			Pages: []PageConfig{
				{
					ID:   "home",
					Path: "/",
					Thresholds: CategoryThresholds{
						"performance":   {Error: 0.1, Warn: 0.5}, // Low thresholds for test page
						"accessibility": {Error: 0.5, Warn: 0.8},
					},
				},
			},
			GlobalOptions: &GlobalOptions{
				TimeoutMs: 90000,
				Lighthouse: &LighthouseSettings{
					Settings: &LighthouseRunnerSettings{
						OnlyCategories:   []string{"performance", "accessibility"},
						ThrottlingMethod: "simulate",
						FormFactor:       "desktop",
					},
				},
			},
		}

		validator := New(ValidatorConfig{
			BaseURL: testServer.URL,
			Config:  config,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		result := validator.Audit(ctx)

		if result.Skipped {
			t.Fatalf("Expected audit to run, but it was skipped: %v", result.Observations)
		}

		if !result.Success {
			t.Logf("Audit completed with failures (may be expected for simple test page)")
			for _, obs := range result.Observations {
				t.Logf("  Observation: %s", obs.Message)
			}
			for _, pr := range result.PageResults {
				t.Logf("  Page %s scores: %v", pr.PageID, pr.Scores)
				for _, v := range pr.Violations {
					t.Logf("    Violation: %s", v.String())
				}
			}
		}

		// Verify we got page results
		if len(result.PageResults) == 0 {
			t.Fatal("Expected at least one page result")
		}

		// Verify scores were captured
		pr := result.PageResults[0]
		if len(pr.Scores) == 0 {
			t.Fatal("Expected scores in page result")
		}

		// Log actual scores for visibility
		t.Logf("Lighthouse scores for test page:")
		for category, score := range pr.Scores {
			t.Logf("  %s: %.0f%%", category, score*100)
		}
	})

	t.Run("Validator_HandlesMultiplePages", func(t *testing.T) {
		config := &Config{
			Enabled: true,
			Pages: []PageConfig{
				{
					ID:   "home",
					Path: "/",
					Thresholds: CategoryThresholds{
						"performance": {Error: 0.1, Warn: 0.5},
					},
				},
				{
					ID:   "about",
					Path: "/about",
					Thresholds: CategoryThresholds{
						"performance": {Error: 0.1, Warn: 0.5},
					},
				},
			},
			GlobalOptions: &GlobalOptions{
				TimeoutMs: 90000,
				Lighthouse: &LighthouseSettings{
					Settings: &LighthouseRunnerSettings{
						OnlyCategories:   []string{"performance"},
						ThrottlingMethod: "simulate",
						FormFactor:       "desktop",
					},
				},
			},
		}

		validator := New(ValidatorConfig{
			BaseURL: testServer.URL,
			Config:  config,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		result := validator.Audit(ctx)

		if result.Skipped {
			t.Fatalf("Expected audit to run, but it was skipped")
		}

		// Should have results for both pages
		if len(result.PageResults) != 2 {
			t.Errorf("Expected 2 page results, got %d", len(result.PageResults))
		}

		// Both pages should have scores
		for _, pr := range result.PageResults {
			if len(pr.Scores) == 0 {
				t.Errorf("Page %s has no scores", pr.PageID)
			}
			t.Logf("Page %s: performance=%.0f%%", pr.PageID, pr.Scores["performance"]*100)
		}
	})
}

// TestIntegration_Validator_MobileViewport tests mobile viewport configuration.
func TestIntegration_Validator_MobileViewport(t *testing.T) {
	checkLighthouseAvailable(t)
	checkChromeAvailable(t)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mobile Test</title>
</head>
<body>
    <h1>Mobile Test Page</h1>
</body>
</html>`)
	}))
	defer testServer.Close()

	config := &Config{
		Enabled: true,
		Pages: []PageConfig{
			{
				ID:       "mobile-home",
				Path:     "/",
				Viewport: "mobile",
				Thresholds: CategoryThresholds{
					"performance": {Error: 0.1, Warn: 0.5},
				},
			},
		},
		GlobalOptions: &GlobalOptions{
			TimeoutMs: 90000,
			Lighthouse: &LighthouseSettings{
				Settings: &LighthouseRunnerSettings{
					OnlyCategories:   []string{"performance"},
					ThrottlingMethod: "simulate",
				},
			},
		},
	}

	validator := New(ValidatorConfig{
		BaseURL: testServer.URL,
		Config:  config,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result := validator.Audit(ctx)

	if result.Skipped {
		t.Fatalf("Expected audit to run, but it was skipped")
	}

	if len(result.PageResults) == 0 {
		t.Fatal("Expected at least one page result")
	}

	pr := result.PageResults[0]
	t.Logf("Mobile page %s: performance=%.0f%%", pr.PageID, pr.Scores["performance"]*100)
}
