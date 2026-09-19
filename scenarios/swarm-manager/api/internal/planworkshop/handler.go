package planworkshop

import (
	"encoding/json"
	"net/http"

	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/plan-workshops", h.Open).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/plan-workshops/{id}", h.Get).Methods(http.MethodGet)
	// Deprecated transition aliases: use TransitionService.StartTransition/ApplyTransition.
	r.HandleFunc("/api/v1/plan-workshops/{id}/review", h.StartReview).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/plan-workshops/{id}/responses", h.SubmitResponse).Methods(http.MethodPost)
}

func (h *Handler) StartReview(w http.ResponseWriter, r *http.Request) {
	session, review, err := h.service.StartReview(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, struct {
		Session Session   `json:"session"`
		Review  ReviewRun `json:"review"`
	}{session, review})
}

func (h *Handler) ApplyReview(w http.ResponseWriter, r *http.Request) {
	session, review, err := h.service.ApplyReview(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, struct {
		Session Session   `json:"session"`
		Review  ReviewRun `json:"review"`
	}{session, review})
}

func (h *Handler) ApplyReconciliation(w http.ResponseWriter, r *http.Request) {
	session, resolution, err := h.service.ApplyReconciliation(r.Context(), mux.Vars(r)["id"], mux.Vars(r)["responseID"])
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, struct {
		Session    Session    `json:"session"`
		Resolution Resolution `json:"resolution"`
	}{session, resolution})
}

func (h *Handler) ApplyCandidate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AcknowledgeQualityImpact bool `json:"acknowledge_quality_impact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid candidate application request")
		return
	}
	session, resolution, err := h.service.ApplyCandidate(r.Context(), mux.Vars(r)["id"], mux.Vars(r)["responseID"], request.AcknowledgeQualityImpact)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, struct {
		Session    Session    `json:"session"`
		Resolution Resolution `json:"resolution"`
	}{session, resolution})
}

func (h *Handler) DiscardCandidate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid candidate discard request")
		return
	}
	session, resolution, err := h.service.DiscardCandidate(r.Context(), mux.Vars(r)["id"], mux.Vars(r)["responseID"], request.Reason)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, struct {
		Session    Session    `json:"session"`
		Resolution Resolution `json:"resolution"`
	}{session, resolution})
}

func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Subject Subject       `json:"subject"`
		Packet  *ReviewPacket `json:"packet,omitempty"`
	}
	if json.NewDecoder(r.Body).Decode(&request) != nil {
		writeError(w, http.StatusBadRequest, "invalid workshop request")
		return
	}
	session, err := h.service.OpenOrGet(request.Subject, request.Packet)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, session)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Get(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "plan workshop not found")
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, session)
}

func (h *Handler) SubmitResponse(w http.ResponseWriter, r *http.Request) {
	var response Response
	if json.NewDecoder(r.Body).Decode(&response) != nil {
		writeError(w, http.StatusBadRequest, "invalid workshop response")
		return
	}
	session, resolution, err := h.service.SubmitResponse(r.Context(), mux.Vars(r)["id"], response)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusOK, struct {
		Session    Session    `json:"session"`
		Resolution Resolution `json:"resolution"`
	}{session, resolution})
}

func writeError(w http.ResponseWriter, status int, message string) {
	_ = httputil.JSONWithStatus(w, status, map[string]string{"error": message})
}
