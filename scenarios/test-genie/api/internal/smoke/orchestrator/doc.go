// Package orchestrator coordinates the UI smoke test workflow.
//
// The Orchestrator is the entry point for running UI smoke tests.
// It coordinates preflight checks, browser execution, handshake detection,
// and artifact persistence.
//
// # Usage
//
//	cfg := smoke.Config{
//	    ScenarioName:   "my-scenario",
//	    ScenarioDir:    "/path/to/scenario",
//	    BrowserlessURL: "http://localhost:4110",
//	    Timeout:        90 * time.Second,
//	    HandshakeTimeout: 15 * time.Second,
//	}
//
//	orch := orchestrator.New(cfg,
//	    orchestrator.WithLogger(os.Stdout),
//	    orchestrator.WithPreflightChecker(preflight.NewChecker(cfg)),
//	    orchestrator.WithBrowserClient(browser.NewClient(cfg.BrowserlessURL)),
//	)
//
//	result, err := orch.Run(ctx)
package orchestrator
