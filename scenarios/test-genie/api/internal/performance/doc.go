// Package performance provides build benchmark and performance validation
// for the test-genie scenario testing framework.
//
// It orchestrates Go API builds, Node.js UI builds, and validates them
// against configurable time thresholds. The package follows a dependency
// injection pattern for testability.
//
// # Architecture
//
// The package is organized into sub-packages:
//   - golang: Go build benchmarking
//   - nodejs: Node.js/UI build benchmarking
//
// Each sub-package defines a Validator interface that can be mocked for testing.
//
// # Usage
//
//	runner := performance.New(performance.Config{
//	    ScenarioDir:  "/path/to/scenario",
//	    ScenarioName: "my-scenario",
//	    Expectations: expectations,
//	}, performance.WithLogger(os.Stdout))
//
//	result := runner.Run(ctx)
//	if !result.Success {
//	    log.Printf("Performance validation failed: %v", result.Error)
//	}
package performance
