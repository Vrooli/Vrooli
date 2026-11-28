package prompts

import (
	"net/http"

	"swarm-manager/internal/apierr"
)

// ExperimentsRetired answers the former experiment results route. Prompt
// experiments no longer exist, so no id resolves.
func (h *Handler) ExperimentsRetired(w http.ResponseWriter, _ *http.Request) {
	apierr.MapError(w, "[experiments] results", apierr.NotFound("prompt experiments are retired"))
}
