// Package lighthouse provides Lighthouse performance auditing via the official CLI.
//
// This package shells out to Google's official Lighthouse CLI to run
// performance, accessibility, best-practices, and SEO audits against
// scenario UI pages. It validates scores against configurable thresholds
// defined in .vrooli/lighthouse.json.
//
// # Prerequisites
//
// The Lighthouse CLI must be available:
//
//	npm install -g lighthouse
//
// Or it will be run via npx if available. Chrome/Chromium must also be
// installed on the system.
//
// # Configuration
//
// Lighthouse audits are configured via .vrooli/lighthouse.json:
//
//	{
//	  "enabled": true,
//	  "pages": [
//	    {
//	      "id": "home",
//	      "path": "/",
//	      "thresholds": {
//	        "performance": { "error": 0.75, "warn": 0.85 },
//	        "accessibility": { "error": 0.90, "warn": 0.95 }
//	      }
//	    }
//	  ]
//	}
//
// # Usage
//
//	cfg, err := lighthouse.LoadConfig(scenarioDir)
//	if err != nil {
//	    return err
//	}
//
//	validator := lighthouse.New(lighthouse.ValidatorConfig{
//	    BaseURL: "http://localhost:3000",
//	    Config:  cfg,
//	})
//
//	result := validator.Audit(ctx)
//	if !result.Success {
//	    // Handle failures
//	}
//
// # Architecture
//
// This package uses Google's official Lighthouse CLI rather than third-party
// wrappers. The CLI is invoked with appropriate flags for JSON output,
// and the results are parsed to extract category scores and audit details.
//
// The CLIRunner type implements the Client interface and handles:
//   - Finding the lighthouse CLI (direct install or via npx)
//   - Building CLI arguments from configuration
//   - Executing audits with proper timeout handling
//   - Parsing JSON output into structured results
package lighthouse
