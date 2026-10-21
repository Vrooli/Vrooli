package photographic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"device-studio/internal/module"
	"device-studio/internal/photographic"
	"device-studio/internal/projects"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Owner interface {
	Submit(context.Context, string, string, int, int) (photographic.Operation, int, error)
	Lookup(context.Context, string) (photographic.Operation, error)
	Wait(context.Context, string) (photographic.Operation, error)
	Cancel(context.Context, string) (photographic.Operation, error)
	PNG(context.Context, string, int, int) ([]byte, error)
}

type Handler struct {
	projects projects.Service
	owner    Owner
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "photographic_submit", Path: "/api/v1/projects/{project_id}/photographic-scenes", Method: "POST", Summary: "Submit one local photographic scene generation", Description: "Browser-owned request adapter to Image Tools; local-only owner job execution remains authoritative.", Category: "projects", Request: &module.Schema{Type: "object", Properties: map[string]string{"operation_id": "UUID", "prompt": "string (<=800 UTF-8 bytes)", "width": "768 or 512", "height": "512 or 768"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "state": "string", "model_id": "string", "tier": "string"}}, RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The project workbench needs a bounded JSON/browser action; Image Tools remains the sole generation and job owner."}},
	{ID: "photographic_read", Path: "/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}", Method: "GET", Summary: "Read local generation status", Description: "Reads an Image Tools operation record without submitting work.", Category: "projects", Response: &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "state": "string", "progress": "integer", "result_ref": "string"}}, RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The browser reads a caller-key status DTO; the durable record remains owned by Image Tools."}},
	{ID: "photographic_wait", Path: "/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}/wait", Method: "POST", Summary: "Wait once for local generation", Description: "Uses the Image Tools block-once WaitJob verb.", Category: "projects", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The browser invokes one block-once owner wait through its project API origin."}},
	{ID: "photographic_cancel", Path: "/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}/cancel", Method: "POST", Summary: "Request local generation cancellation", Description: "Forwards cancellation to the Image Tools job owner and returns its current state.", Category: "projects", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The browser invokes the owner cancellation through its project API origin."}},
	{ID: "photographic_content", Path: "/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}/content", Method: "GET", Summary: "Read verified generated PNG", Description: "Copies a successful managed Image Tools PNG after checking type, size, and requested dimensions.", Category: "projects", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "Binary owner content is verified and proxied through the project API; callers cannot provide filesystem paths."}},
}

func Module(projects projects.Service, owner Owner) module.Module {
	h := &Handler{projects: projects, owner: owner}
	return module.Module{Name: "photographic", Endpoints: Endpoints, Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/projects/{project_id}/photographic-scenes", h.submit).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}", h.lookup).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}/wait", h.wait).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}/cancel", h.cancel).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/projects/{project_id}/photographic-scenes/operations/{operation_id}/content", h.content).Methods(http.MethodGet)
	}}
}

type request struct {
	OperationID string `json:"operation_id"`
	Prompt      string `json:"prompt"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

func (h *Handler) project(w http.ResponseWriter, r *http.Request) bool {
	if _, err := h.projects.Get(r.Context(), mux.Vars(r)["project_id"]); err != nil {
		var missing projects.ErrProjectNotFound
		if errors.As(err, &missing) {
			writeError(w, http.StatusNotFound, "project not found")
		} else {
			writeError(w, http.StatusInternalServerError, "project could not be checked")
		}
		return false
	}
	return true
}
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	if !h.project(w, r) {
		return
	}
	var in request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if _, err := uuid.Parse(in.OperationID); err != nil {
		writeError(w, http.StatusBadRequest, "operation_id must be a UUID")
		return
	}
	if strings.TrimSpace(in.Prompt) == "" || len([]byte(in.Prompt)) > 800 || strings.ContainsRune(in.Prompt, '\x00') {
		writeError(w, http.StatusBadRequest, "prompt must contain 1 to 800 UTF-8 bytes")
		return
	}
	if !((in.Width == 768 && in.Height == 512) || (in.Width == 512 && in.Height == 768)) {
		writeError(w, http.StatusBadRequest, "unsupported output dimensions")
		return
	}
	op, status, err := h.owner.Submit(r.Context(), in.OperationID, in.Prompt, in.Width, in.Height)
	if err != nil {
		if status == 0 {
			status = http.StatusServiceUnavailable
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, status, op)
}
func (h *Handler) lookup(w http.ResponseWriter, r *http.Request) {
	if !h.project(w, r) {
		return
	}
	op, err := h.owner.Lookup(r.Context(), mux.Vars(r)["operation_id"])
	if err != nil {
		writeError(w, ownerReadStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, op)
}
func (h *Handler) wait(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	if !h.project(w, r) {
		return
	}
	op, err := h.owner.Lookup(r.Context(), mux.Vars(r)["operation_id"])
	if err != nil {
		writeError(w, ownerReadStatus(err), err.Error())
		return
	}
	op, err = h.owner.Wait(r.Context(), op.JobID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "job wait did not return an authoritative state")
		return
	}
	writeJSON(w, http.StatusOK, op)
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if !h.project(w, r) {
		return
	}
	op, err := h.owner.Lookup(r.Context(), mux.Vars(r)["operation_id"])
	if err != nil {
		writeError(w, ownerReadStatus(err), err.Error())
		return
	}
	op, err = h.owner.Cancel(r.Context(), op.JobID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "job cancellation could not be confirmed")
		return
	}
	writeJSON(w, http.StatusOK, op)
}
func (h *Handler) content(w http.ResponseWriter, r *http.Request) {
	if !h.project(w, r) {
		return
	}
	op, err := h.owner.Lookup(r.Context(), mux.Vars(r)["operation_id"])
	if err != nil {
		writeError(w, ownerReadStatus(err), err.Error())
		return
	}
	if op.State != "succeeded" {
		writeError(w, http.StatusConflict, "generated image is not available until the job succeeds")
		return
	}
	width, ew := strconv.Atoi(r.URL.Query().Get("width"))
	height, eh := strconv.Atoi(r.URL.Query().Get("height"))
	if ew != nil || eh != nil {
		writeError(w, http.StatusBadRequest, "requested dimensions are required")
		return
	}
	data, err := h.owner.PNG(r.Context(), op.ResultRef, width, height)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}
func ownerReadStatus(err error) int {
	var ownerErr photographic.OwnerHTTPError
	if errors.As(err, &ownerErr) && ownerErr.Status == http.StatusNotFound {
		return http.StatusNotFound
	}
	return http.StatusBadGateway
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprint(message)})
}
