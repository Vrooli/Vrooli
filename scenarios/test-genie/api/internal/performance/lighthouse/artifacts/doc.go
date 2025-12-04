// Package artifacts provides file writing functionality for Lighthouse audit results.
//
// The Writer interface allows saving audit results in multiple formats:
//   - Per-page JSON reports with full Lighthouse data
//   - Phase results JSON for integration with the business phase
//   - Summary JSON with aggregated results across all pages
//
// Reports are written to the scenario's coverage directory:
//   - coverage/lighthouse/     - Per-page reports and summary
//   - coverage/phase-results/  - Phase integration results
package artifacts
