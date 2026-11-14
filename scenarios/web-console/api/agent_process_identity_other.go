//go:build !linux

package main

// platformDiscoverAgentProcesses reports nothing on hosts without a supported
// process-inspection mechanism. Identification then relies on the agent's own
// hook, which is the behavior these platforms already had — the fallback adds
// resilience where it can and never removes any.
func platformDiscoverAgentProcesses() ([]agentProcess, error) { return nil, nil }
