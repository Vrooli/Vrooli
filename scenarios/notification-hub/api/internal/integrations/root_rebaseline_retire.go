package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"notification-hub/internal/hub"
)

const RootRebaselineRetireRoute = "/api/v1/internal/root-rebaseline/retire"

type rootRebaselineRetirer interface {
	RetireRootRebaseline(context.Context, string, string) error
}

// RootRebaselineRetireHandler seals an expired unconsumed decision; it does not
// disclose or alter a consumed approval that may still be needed for recovery.
func RootRebaselineRetireHandler(store rootRebaselineRetirer, verifier authn.TokenVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if store == nil || verifier == nil {
			http.Error(w, "decision service unavailable", http.StatusServiceUnavailable)
			return
		}
		principal, err := providerDecisionPrincipal(r, verifier)
		if err != nil || !principal.Verified || principal.Kind != apiidentity.ActorService || principal.Subject != RootRebaselineSubject || principal.Audience != RootRebaselineAudience || !slices.Contains(principal.Scopes, RootRebaselineScope) {
			http.Error(w, "service identity required", http.StatusUnauthorized)
			return
		}
		var in rootRebaselineRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil {
			http.Error(w, "invalid decision request", http.StatusBadRequest)
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) || in.RequesterRef == "" || strings.TrimSpace(in.RequesterRef) != in.RequesterRef || len(in.RequesterRef) > 128 || !validProviderDecisionDigest(in.Digest) || in.Version != 0 || in.Operation != "" || in.Deadline != "" || in.CandidateSHA != "" || in.ServingSHA != "" || in.WatchdogSHA != "" || in.BundleSHA != "" || in.CandidateContract != "" || in.ServingContract != "" || len(in.Changes) != 0 {
			http.Error(w, "invalid decision request", http.StatusBadRequest)
			return
		}
		if err := store.RetireRootRebaseline(r.Context(), in.RequesterRef, in.Digest); err != nil {
			if errors.Is(err, hub.ErrRootDecisionUnavailable) {
				http.Error(w, "decision unavailable", http.StatusConflict)
			} else {
				http.Error(w, "decision service unavailable", http.StatusServiceUnavailable)
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"requester_ref": in.RequesterRef, "canonical_terms_sha256": in.Digest, "state": "retired"})
	})
}
