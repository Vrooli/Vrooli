package effortauthority

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

const ConsentEndpoint = "/auth/finite-effort"

// ConsentHandler consumes issuer-vetted proposals, never client policy bodies.
// Main does not mount it implicitly. The configured same-origin cookie belongs
// to the existing identity provider; no bearer is returned or refreshed.
type ConsentHandler struct {
	Authority  *Authority
	Proposals  map[string]Policy
	Origin     string
	CookieName string
}

var consentPage = template.Must(template.New("consent").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Approve finite work</title></head><body><main><h1>Approve finite work</h1><p>Review the complete immutable proposal. This permission survives sign-out until its deadline. Revoking it prevents later admission; it does not stop already admitted runs.</p><pre>{{.Policy}}</pre><form method="post" action="/auth/finite-effort"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="digest" value="{{.Digest}}"><label><input type="checkbox" name="surviveLogout" value="yes" required>I approve this exact proposal and understand it survives sign-out.</label><button name="action" value="approve">Approve finite work</button><button name="action" value="revoke" formnovalidate>Revoke finite work</button></form></main></body></html>`))

func (h *ConsentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	refuse := func() { http.Error(w, "finite consent refused", http.StatusForbidden) }
	if h == nil || h.Authority == nil || h.Authority.Approver == nil || h.Authority.Owners == nil || h.Authority.Scope == nil || h.CookieName == "" || r.URL.Path != ConsentEndpoint || (r.Method != "GET" && r.Method != "POST") {
		refuse()
		return
	}
	origin, e := url.Parse(h.Origin)
	if e != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || r.TLS == nil || r.Host != origin.Host {
		refuse()
		return
	}
	if len(r.Header.Values("Authorization")) != 0 || len(r.Header.Values("X-Agent-Identity-Token")) != 0 || len(r.Header.Values(Header)) != 0 {
		refuse()
		return
	}
	cookies := r.Cookies()
	token := ""
	cookieCount := 0
	for _, cookie := range cookies {
		if cookie.Name == h.CookieName {
			cookieCount++
			if cookieCount != 1 {
				refuse()
				return
			}
			token = cookie.Value
		}
	}
	if token == "" {
		refuse()
		return
	}
	owner, e := h.Authority.Approver.ApproveEffortOwner(r.Context(), token)
	if e != nil || !owner.Enabled {
		refuse()
		return
	}
	id := r.URL.Query().Get("id")
	if r.Method == "POST" {
		if r.URL.RawQuery != "" || r.Header.Get("Origin") != h.Origin || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			refuse()
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if r.ParseForm() != nil || len(r.PostForm["id"]) != 1 || len(r.PostForm["digest"]) != 1 || len(r.PostForm["action"]) != 1 {
			refuse()
			return
		}
		for key := range r.PostForm {
			if key != "id" && key != "digest" && key != "action" && key != "surviveLogout" {
				refuse()
				return
			}
		}
		id = r.PostForm.Get("id")
	}
	p, exists := h.Proposals[id]
	if !exists || p.Owner != owner.Subject || p.Validate() != nil {
		refuse()
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == "GET" {
		raw, e := json.MarshalIndent(p, "", "  ")
		if e != nil {
			refuse()
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = consentPage.Execute(w, struct{ ID, Digest, Policy string }{p.ID, Digest(p), string(raw)})
		return
	}
	if r.PostForm.Get("digest") != Digest(p) {
		refuse()
		return
	}
	switch r.PostForm.Get("action") {
	case "approve":
		if len(r.PostForm["surviveLogout"]) != 1 || r.PostForm.Get("surviveLogout") != "yes" {
			refuse()
			return
		}
		e = h.Authority.Approve(r.Context(), token, p)
	case "revoke":
		e = h.Authority.Revoke(r.Context(), token, p.ID)
	default:
		e = ErrRefused
	}
	if e != nil {
		refuse()
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Finite work decision recorded.\n"))
}
