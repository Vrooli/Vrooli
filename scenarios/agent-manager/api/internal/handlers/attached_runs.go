package handlers

import (
	"io"
	"net/http"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// AttachRun creates an identity-bearing run for a harness session that was
// started outside agent-manager. It never accepts a transcript path or live
// output stream.
func (h *Handler) AttachRun(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}
	var protoReq apipb.AttachRunRequest
	if len(body) > 0 {
		if err := protoconv.UnmarshalJSON(body, &protoReq); err != nil {
			writeSimpleError(w, r, "body", "invalid JSON request body: "+err.Error())
			return
		}
	}
	if !h.validateProto(w, r, &protoReq) {
		return
	}

	var taskID *uuid.UUID
	if protoReq.TaskId != nil {
		id, err := uuid.Parse(protoReq.GetTaskId())
		if err != nil {
			writeSimpleError(w, r, "task_id", "invalid UUID format for task ID")
			return
		}
		taskID = &id
	}
	processID := 0
	if protoReq.ProcessId != nil {
		processID = int(protoReq.GetProcessId())
	}
	result, err := h.svc.AttachRun(r.Context(), orchestration.AttachRunRequest{
		TaskID:         taskID,
		HarnessKind:    protoReq.GetHarnessKind(),
		HarnessSession: protoReq.GetHarnessSessionId(),
		ProcessID:      processID,
		HarnessTitle:   protoReq.GetHarnessTitle(),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusCreated, &apipb.AttachRunResponse{
		Run:           protoconv.RunToProto(result.Run),
		IdentityToken: result.Token,
		ExpiresAt:     protoconv.TimestampToProto(result.ExpiresAt),
	})
}

// DetachRun closes an attached run without signaling or terminating the
// harness process.
func (h *Handler) DetachRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(muxVarsID(r))
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	request := apipb.DetachRunRequest{RunId: id.String()}
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		writeSimpleError(w, r, "body", "failed to read request body")
		return
	}
	if len(body) > 0 {
		if err := protoconv.UnmarshalJSON(body, &request); err != nil {
			writeSimpleError(w, r, "body", "invalid JSON request body: "+err.Error())
			return
		}
		request.RunId = id.String()
	}
	if !h.validateProto(w, r, &request) {
		return
	}
	run, err := h.svc.DetachRun(r.Context(), id, request.GetReason())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, &apipb.DetachRunResponse{Run: protoconv.RunToProto(run)})
}

func muxVarsID(r *http.Request) string {
	// Keep this tiny adapter local to the attached surface so it remains easy
	// to audit that the path, not the request body, chooses the run identity.
	return mux.Vars(r)["id"]
}
