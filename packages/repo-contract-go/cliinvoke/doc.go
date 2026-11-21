// Package cliinvoke is the only place that resolves, runs, and classifies a
// `vrooli` CLI subprocess.
//
// Every long-lived supervisor (the autoheal loop, the runtime supervisor, the
// autoheal API's check executor, the scenario-CLI preflight) is a consumer of
// the CLI's argv contract. Before this package each of them found the binary
// its own way and read failures its own way, and only one of them carried the
// timeout and WaitDelay discipline that the 2026-08-01 inherited-pipe outage
// taught. The package holds that discipline once.
//
// The package lives inside repo-contract-go and imports only the standard
// library on purpose: every module in the repository already replaces
// repo-contract-go, so no consumer needs a new go.mod line, and the autoheal
// loop, which refuses the api-core and proto dependency graph, can use it
// without pulling either in.
package cliinvoke
