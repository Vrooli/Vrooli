package pipeline

// Handler owns pipeline domain dependencies for the generated Connect service.
// It deliberately has no HTTP route registration: every pipeline operation is
// represented by PipelineService in the scenario proto contract.
type Handler struct {
	orchestrator Orchestrator
	manager      *Manager
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithOrchestrator sets the orchestration seam.
func WithOrchestrator(orchestrator Orchestrator) HandlerOption {
	return func(handler *Handler) {
		handler.orchestrator = orchestrator
	}
}

// WithManager sets the active-pipeline management seam.
func WithManager(manager *Manager) HandlerOption {
	return func(handler *Handler) {
		handler.manager = manager
	}
}

// NewHandler constructs the domain dependency holder consumed by ConnectService.
func NewHandler(options ...HandlerOption) *Handler {
	handler := &Handler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}
