package state

// Handler owns state behavior consumed by the generated StateService.
// The previous REST administration surface was retired; all state operations
// now use the typed Connect contract.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}
