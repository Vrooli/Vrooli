package handlers

import (
	"net/http"

	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

// GetRunAccounting returns one run's metered usage so a consumer that
// dispatched a standalone run can settle its reservation.
func (h *Handler) GetRunAccounting(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	accounting, err := h.svc.RunAccounting(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	// Zero and false are meaningful here: unknown usage must be visible as
	// tokens_known=false, not as an omitted field a reader could default.
	data, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(runAccountingToProto(accounting))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to serialize run accounting")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func runAccountingToProto(a orchestration.RunAccounting) *apipb.RunAccounting {
	return &apipb.RunAccounting{
		RunId:          a.RunID.String(),
		Terminal:       a.Terminal,
		Tokens:         a.Tokens,
		Turns:          a.Turns,
		TokensKnown:    a.TokensKnown,
		ChargeMicroUsd: a.ChargeMicroUSD,
		ChargeMeasured: a.ChargeMeasured,
		WallSeconds:    a.WallSeconds,
	}
}
