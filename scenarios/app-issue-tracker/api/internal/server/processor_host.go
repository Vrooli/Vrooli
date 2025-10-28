package server

// processorHost is an adapter that implements automation.Host interface for the Processor.
//
// Design rationale (Interface Segregation Principle):
// The automation.Processor lives in a separate package and should not depend on the entire
// Server struct (which has dozens of methods and fields). Instead, we define a minimal
// automation.Host interface with only the 4 methods the processor actually needs.
//
// This approach provides several benefits:
// 1. Clear dependency boundary - processor only knows about the operations it uses
// 2. Easier testing - can mock just the Host interface instead of entire Server
// 3. Prevents circular dependencies between packages
// 4. Makes processor portable - could be reused with different server implementations
//
// Alternative considered: Having Server implement automation.Host directly
// Rejected because: Would couple automation package to server package structure
type processorHost struct {
	server *Server
}

func newProcessorHost(server *Server) *processorHost {
	return &processorHost{server: server}
}

func (h *processorHost) ClearExpiredRateLimitMetadata() {
	h.server.investigations.ClearExpiredRateLimitMetadata()
}

func (h *processorHost) LoadIssuesFromFolder(folder string) ([]Issue, error) {
	return h.server.loadIssuesFromFolder(folder)
}

func (h *processorHost) ClearRateLimitMetadata(issueID string) {
	h.server.investigations.ClearRateLimitMetadata(issueID)
}

func (h *processorHost) TriggerInvestigation(issueID, agentID string, autoResolve bool) error {
	return h.server.investigations.TriggerInvestigation(issueID, agentID, autoResolve)
}

func (h *processorHost) CleanupOldTranscripts() {
	h.server.investigations.CleanupOldTranscripts()
}
