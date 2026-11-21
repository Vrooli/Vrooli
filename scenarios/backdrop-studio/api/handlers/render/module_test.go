package render

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	renderconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/render/render_v1connect"
)

// TestSubmitCarriesTheRenderBudget pins the deadline a render runs under.
//
// A CPU diffusion render takes tens of minutes; under the ordinary request
// deadline it was cut off within seconds and its image-tools job orphaned.
// Submit alone gets the long budget — the read procedures keep the server's
// ordinary deadline, so a slow GetJob still fails fast.
func TestSubmitCarriesTheRenderBudget(t *testing.T) {
	deadlines := map[string]time.Time{}
	server := httptest.NewServer(budgetSubmit(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, _ := r.Context().Deadline()
		deadlines[r.URL.Path] = deadline
	})))
	defer server.Close()

	for _, procedure := range []string{renderconnect.RenderServiceSubmitProcedure, renderconnect.RenderServiceGetJobProcedure} {
		resp, err := server.Client().Post(server.URL+procedure, "application/json", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	}

	require.WithinDuration(t, time.Now().Add(renderSubmitBudget), deadlines[renderconnect.RenderServiceSubmitProcedure], time.Minute,
		"Submit must run under the render budget")
	require.GreaterOrEqual(t, renderSubmitBudget, time.Hour, "the budget must outlast a CPU diffusion render")
	require.True(t, deadlines[renderconnect.RenderServiceGetJobProcedure].IsZero(), "only Submit gets the render budget")
}
