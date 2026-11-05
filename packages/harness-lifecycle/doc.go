// Package agentinstall owns the shared coding-harness lifecycle boundary.
// Full deployments use it through control-plane resource actions and the
// Dependency Analyzer; desktop uses verified native acquisition. Both prepare
// declared compressed native artifacts without running installer scripts.
package agentinstall
