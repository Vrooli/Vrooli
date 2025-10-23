package server

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
