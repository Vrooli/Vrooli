package handlers

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type AuthorizationService interface {
	RequestAuthorization(context.Context, orchestration.AuthorizationRequest) (*domain.AuthorizationGrant, error)
	GetAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error)
	ListAuthorizations(context.Context, int) ([]*domain.AuthorizationGrant, error)
	ApproveAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error)
	RevokeAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error)
}

func RegisterAuthorizations(r *mux.Router, svc AuthorizationService) {
	if svc == nil {
		return
	}
	r.HandleFunc("/api/v1/authorizations/requests", func(w http.ResponseWriter, r *http.Request) {
		// Requesting consent has no authority and never attaches or starts a run.
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
			http.Error(w, "local control-plane request required", http.StatusForbidden)
			return
		}
		var req orchestration.AuthorizationRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid authorization request", http.StatusBadRequest)
			return
		}
		grant, err := svc.RequestAuthorization(r.Context(), req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, grant)
	}).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/authorizations", func(w http.ResponseWriter, r *http.Request) {
		host, _, remoteErr := net.SplitHostPort(r.RemoteAddr)
		if _, err := authn.RequireHuman(r.Context()); err != nil && (remoteErr != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback()) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		pid, err := strconv.Atoi(r.URL.Query().Get("pid"))
		if err != nil || pid <= 1 {
			http.Error(w, "PID required", http.StatusBadRequest)
			return
		}
		grants, err := svc.ListAuthorizations(r.Context(), pid)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, grants)
	}).Methods(http.MethodGet)
	for _, action := range []string{"approve", "revoke"} {
		action := action
		r.HandleFunc("/api/v1/authorizations/{id}/"+action, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if len(r.Header.Values("X-Agent-Identity-Token")) != 0 {
				http.Error(w, "agent credentials cannot approve or revoke owner consent", http.StatusForbidden)
				return
			}
			if _, err := authn.RequireHuman(r.Context()); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			id, err := uuid.Parse(mux.Vars(r)["id"])
			if err != nil {
				http.Error(w, "invalid authorization identifier", http.StatusBadRequest)
				return
			}
			if r.Method == http.MethodGet {
				grant, err := svc.GetAuthorization(r.Context(), id)
				if err != nil {
					writeError(w, r, err)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				consentTemplate.Execute(w, struct {
					Grant  *domain.AuthorizationGrant
					Action string
				}{grant, action})
				return
			}
			// Browser approval is same-origin. Native clients need independently
			// verified human bearer proof; there is no local-UID exchange here.
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host || (u.Scheme != "https" && u.Scheme != "http") {
					http.Error(w, "same-origin consent required", http.StatusForbidden)
					return
				}
			} else if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				http.Error(w, "browser Origin or human bearer proof required", http.StatusForbidden)
				return
			}
			var grant *domain.AuthorizationGrant
			if action == "approve" {
				grant, err = svc.ApproveAuthorization(r.Context(), id)
			} else {
				grant, err = svc.RevokeAuthorization(r.Context(), id)
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			writeJSON(w, http.StatusOK, grant)
		}).Methods(http.MethodGet, http.MethodPost)
	}
}

var consentTemplate = template.Must(template.New("consent").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Agent authorization</title><main><h1>Agent authorization</h1><p>Decision: {{.Action}}</p><p>Policy: {{.Grant.Policy}}</p><p>Session: {{.Grant.HarnessKind}} / {{.Grant.HarnessSession}}</p><p>Process: {{.Grant.Binding.PID}}; state: {{.Grant.State}}</p><p>Duration: {{.Grant.DurationSeconds}} seconds</p><ul>{{range .Grant.Targets}}<li>{{.}}</li>{{end}}</ul><p>This grants an agent authority to perform the listed routine operations. Your identity remains the approver.</p><form method="post"><button type="submit">Confirm {{.Action}}</button></form></main></html>`))
