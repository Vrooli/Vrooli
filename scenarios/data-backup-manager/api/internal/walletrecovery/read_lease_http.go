package walletrecovery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// HumanSessionVerifier is the standard request-bound verifier seam. Production
// supplies api-core's Scenario Authenticator JWT verifier; tests can exercise
// the authorization policy without standing up another identity system.
type HumanSessionVerifier interface {
	VerifyRequest(context.Context, *http.Request) (identity.Principal, error)
}

// NewDBMReadLeaseVerifier binds the dedicated lease ingress to exactly one
// relying-party audience. A token shared with any other DBM or scenario API
// must not authorize custody snapshot reservations.
func NewDBMReadLeaseVerifier() *authn.JWTVerifier {
	return newDBMReadLeaseVerifier(authn.JWTConfig{})
}

func newDBMReadLeaseVerifier(cfg authn.JWTConfig) *authn.JWTVerifier {
	cfg.Audience = "scenario-authenticator:data-backup-manager"
	cfg.ExclusiveAudience = true
	return authn.NewScenarioAuthenticatorProvider(cfg)
}

// VerifiedHumanAuthorizer admits only a verified operator session for the
// explicit custody read-lease operations. It does not accept forwarded agent
// identity headers or service principals.
type VerifiedHumanAuthorizer struct {
	Verifier HumanSessionVerifier
}

func (a VerifiedHumanAuthorizer) AuthorizeWalletRecovery(r *http.Request) error {
	if a.Verifier == nil || r == nil {
		return errors.New("verified DBM operator session is not configured")
	}
	principal, err := a.Verifier.VerifyRequest(r.Context(), r)
	if err != nil {
		return errors.New("DBM operator session could not be verified")
	}
	if !principal.IsHuman() {
		return errors.New("wallet recovery read leases require a verified human operator")
	}
	return nil
}

type ReadLeaseRequest struct {
	LeaseID       string `json:"lease_id"`
	TargetID      string `json:"target_id"`
	TargetName    string `json:"target_name,omitempty"`
	DestinationID string `json:"destination_id,omitempty"`
	SnapshotID    string `json:"snapshot_id,omitempty"`
}

type ReadLeaseHTTP struct {
	Service ReadLeaseService
	Auth    Authorizer
}

func (h ReadLeaseHTTP) Handler(action string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if h.Auth == nil {
			http.Error(w, "wallet recovery read lease is not configured", http.StatusServiceUnavailable)
			return
		}
		if err := h.Auth.AuthorizeWalletRecovery(r); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req ReadLeaseRequest
		if err := decodeReadLeaseRequest(w, r, &req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		var result any
		var err error
		switch action {
		case "acquire":
			result, err = h.Service.Acquire(r.Context(), req.LeaseID, req.TargetName, req.DestinationID, req.SnapshotID)
		case "renew":
			result, err = h.Service.Renew(r.Context(), req.LeaseID, req.TargetID)
		case "release":
			err = h.Service.Release(r.Context(), req.LeaseID, req.TargetID)
			result = map[string]string{"lease_id": req.LeaseID, "state": "released"}
		default:
			http.Error(w, "unknown wallet recovery lease action", http.StatusNotFound)
			return
		}
		if err != nil {
			status := http.StatusConflict
			if errors.Is(err, ErrSnapshotSuperseded) {
				status = http.StatusConflict
			}
			http.Error(w, "wallet recovery lease refused: "+err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	})
}

func decodeReadLeaseRequest(w http.ResponseWriter, r *http.Request, out *ReadLeaseRequest) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON object")
	}
	return nil
}
