package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
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
	r.HandleFunc("/api/v1/authorizations/identity", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		p, err := authn.RequireHuman(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"subject": p.Subject, "source": p.Source, "scopes": p.Scopes, "expires_at": p.ExpiresAt})
	}).Methods(http.MethodGet)
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
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "exactly one request object required", http.StatusBadRequest)
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
		now := time.Now()
		for _, g := range grants {
			g.State = g.EffectiveState(now)
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
				grant.State = grant.EffectiveState(time.Now())
				page := consentView(grant, action, "")
				if action == "approve" && page.Open {
					page.MissingCapabilities, err = orchestration.MissingAuthorizationCapabilities(r.Context(), grant)
					if err != nil {
						writeError(w, r, err)
						return
					}
					page.Open = len(page.MissingCapabilities) == 0
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				consentTemplate.Execute(w, page)
				return
			}
			// Browser approval is same-origin. Native clients need independently
			// verified human bearer proof; there is no local-UID exchange here.
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				u, err := url.Parse(origin)
				if err != nil || !authorizationOriginAllowed(u, r) || (u.Scheme != "https" && u.Scheme != "http") {
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
			browser := origin != "" && strings.Contains(r.Header.Get("Accept"), "text/html")
			if err != nil {
				if browser {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusForbidden)
					consentResultTemplate.Execute(w, consentPage{Action: action, Title: "Not " + action + "d", Detail: err.Error()})
					return
				}
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			if browser {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				consentResultTemplate.Execute(w, consentView(grant, action, "Done"))
				return
			}
			writeJSON(w, http.StatusOK, grant)
		}).Methods(http.MethodGet, http.MethodPost)
	}
}

// consentRule is one reviewed policy as the owner reviews it.
type consentRule struct {
	Policy     string
	Operations []string
	Targets    []string
}

type consentPage struct {
	Grant               *domain.AuthorizationGrant
	Action              string
	Title               string
	Rules               []consentRule
	Open                bool
	Detail              string
	MissingCapabilities []string
}

func consentView(g *domain.AuthorizationGrant, action, title string) consentPage {
	page := consentPage{Grant: g, Action: action, Title: title}
	for _, rule := range g.PolicyRules() {
		policy, _ := authn.InteractivePolicy(rule.Policy)
		page.Rules = append(page.Rules, consentRule{Policy: rule.Policy, Operations: policy.Operations, Targets: rule.Targets})
	}
	page.Open = (action == "approve" && g.State == domain.AuthorizationPending) || (action == "revoke" && g.State == domain.AuthorizationApproved)
	return page
}

const consentStyle = `<style>body{font:16px/1.45 system-ui,sans-serif;margin:0;padding:16px;max-width:40rem;color:#111;background:#fff}h1{font-size:1.4rem}section{border:1px solid #ccc;border-radius:8px;padding:8px 12px;margin:8px 0}code{word-break:break-all}button{font-size:1rem;padding:10px 18px}small{color:#555}@media (prefers-color-scheme:dark){body{color:#eee;background:#111}section{border-color:#444}small{color:#aaa}}</style>`

var consentTemplate = template.Must(template.New("consent").Funcs(template.FuncMap{"permissionLifetime": permissionLifetime}).Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Agent authorization</title>` + consentStyle + `<main><h1>Agent authorization: {{.Action}}</h1><p>State: <strong>{{.Grant.State}}</strong>{{if .Grant.SupersededBy}} (replaced by <code>{{.Grant.SupersededBy}}</code>){{end}}</p><p>Session: {{.Grant.HarnessKind}} / <code>{{.Grant.HarnessSession}}</code></p>{{range .Rules}}<section><p>Policy: <strong>{{.Policy}}</strong></p><p>Operations: {{range $i, $o := .Operations}}{{if $i}}, {{end}}<code>{{$o}}</code>{{end}}</p><p>Targets:</p><ul>{{range .Targets}}<li><code>{{.}}</code></li>{{end}}</ul></section>{{end}}<p>Permission lifetime: {{permissionLifetime .Grant.DurationSeconds}}, starting when you confirm.</p><p>Approval deadline: {{.Grant.RequestExpiresAt.Format "2006-01-02 15:04 MST"}}</p><p><small>Process {{.Grant.Binding.PID}}; boundary <code>{{.Grant.Binding.Cgroup}}</code>; boot {{.Grant.Binding.Boot}}; incarnation {{.Grant.Binding.Start}}</small></p><p>This grants an agent authority to perform the listed routine operations. Your identity remains the approver.</p>{{if .MissingCapabilities}}<h2>Approval access needs setup</h2><p>Your sign-in is verified, but it lacks the following capabilities required to approve this request:</p><ul>{{range .MissingCapabilities}}<li><code>{{.}}</code></li>{{end}}</ul><p>Have the verified owner’s capability policy updated, then reload this page to review and confirm. This request remains pending; no permission has been granted.</p>{{else if .Open}}<form method="post"><button type="submit">Confirm {{.Action}}</button></form>{{else}}<p>This request can no longer be {{.Action}}d.</p>{{end}}</main></html>`))

var consentResultTemplate = template.Must(template.New("consent-result").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Agent authorization</title>` + consentStyle + `<main>{{if .Grant}}<h1>{{.Action}}d</h1><p>State: <strong>{{.Grant.State}}</strong></p>{{range .Rules}}<section><p><strong>{{.Policy}}</strong>: {{range $i, $t := .Targets}}{{if $i}}, {{end}}<code>{{$t}}</code>{{end}}</p></section>{{end}}{{if .Grant.Grant}}<p>Expires {{.Grant.Grant.ExpiresAt.Format "2006-01-02 15:04 MST"}}.</p>{{end}}<p>You can close this page.</p>{{else}}<h1>{{.Title}}</h1><p>{{.Detail}}</p>{{end}}</main></html>`))

func permissionLifetime(seconds int64) string {
	if seconds >= 3600 && seconds%3600 == 0 {
		return fmt.Sprintf("%d hours", seconds/3600)
	}
	if seconds >= 60 && seconds%60 == 0 {
		return fmt.Sprintf("%d minutes", seconds/60)
	}
	return fmt.Sprintf("%d seconds", seconds)
}

func authorizationOriginAllowed(origin *url.URL, r *http.Request) bool {
	if origin == nil || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
		return false
	}
	configured := strings.TrimSpace(os.Getenv("AGENT_MANAGER_AUTHORIZATION_URL"))
	if configured != "" {
		allowed, err := url.Parse(configured)
		return err == nil && allowed.Host != "" && allowed.Host == origin.Host && allowed.Scheme == origin.Scheme
	}
	return origin.Host == r.Host
}
