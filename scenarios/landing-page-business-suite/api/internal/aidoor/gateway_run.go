package aidoor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite-api/internal/commerce"
)

const runCallerAssertionHeader = "X-Vrooli-LPBS-Run-Assertion"

type publicRunResult struct {
	Status         string          `json:"status"`
	ValueJSON      json.RawMessage `json:"value_json,omitempty"`
	Validated      bool            `json:"validated,omitempty"`
	ChargedCredits *int64          `json:"charged_credits,omitempty"`
	Error          string          `json:"error,omitempty"`
}

// run forwards a typed app-owned operation through Gateway Run. The role is
// admitted only when the app owns that role in the compiled credit registry;
// the signed assertion then binds the same role and durable operation root.
func (d *Door) run(w http.ResponseWriter, r *http.Request, key Key) {
	if key.Kind != "device" || key.Harness != "" || key.Model != "" || !validCreditKeyScope(key) {
		fail(w, http.StatusForbidden, "an app-scoped device key is required")
		return
	}
	var message inferencev1.RunRequest
	raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err == nil && len(raw) > 1<<20 {
		err = errors.New("request body exceeds limit")
	}
	if err != nil || (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, &message) != nil {
		fail(w, http.StatusBadRequest, "invalid typed inference request")
		return
	}
	if strings.TrimSpace(message.GetRole()) == "" || strings.TrimSpace(message.GetPrivacyProvenance()) == "" || message.GetPrivacyClass() == 0 {
		fail(w, http.StatusBadRequest, "role, privacy class and privacy provenance are required")
		return
	}
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "credit feature owners are unavailable")
		return
	}
	owner, ok := owners.OwnerForRole(message.GetRole())
	if !ok || owner.AppKey != key.App {
		fail(w, http.StatusForbidden, "Gateway role is outside the key app scope")
		return
	}
	if d.SignCreditAssertion == nil {
		fail(w, http.StatusServiceUnavailable, "credit authority is unavailable")
		return
	}
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = randomID()
	}
	operation := callerOperationID(key, idempotencyKey)
	w.Header().Set("X-Operation-ID", operation)
	message.RootOperationId = operation
	requestDigest, err := gatewayRunDigest(&message)
	if err != nil {
		fail(w, http.StatusBadRequest, "typed inference request is invalid")
		return
	}
	if err := d.Keys.beginGatewayRun(r.Context(), operation, key, owner.Feature.ID, message.GetRole(), requestDigest); err != nil {
		fail(w, http.StatusConflict, "operation was already used with a different request")
		return
	}
	assertionFeature := "ai-gateway.run/" + message.GetRole()
	assertion, err := d.SignCreditAssertion(key, operation, assertionFeature)
	if err != nil || strings.TrimSpace(assertion) == "" {
		fail(w, http.StatusServiceUnavailable, "credit authority is unavailable")
		return
	}
	tier, _, err := d.Wallet.AccountCreditScope(r.Context(), key.Account, key.App)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "account credit scope unavailable")
		return
	}
	flow := d.requestLifecycle(operation, requestAccepted)
	defer flow.finish()
	if err := flow.advance(requestPlan); err != nil {
		fail(w, http.StatusInternalServerError, "request state unavailable")
		return
	}
	_, created, err := d.Wallet.ReserveScopedCreditsWithAdmission(r.Context(), key.Account, key.Actor, tier, "ai_credits", key.App, key.MaxCredits, operation, &commerce.ReservationAdmission{
		Reserve: func(ctx context.Context, tx *sql.Tx) error {
			return d.Keys.ReserveCap(ctx, tx, key, operation, key.MaxCredits)
		},
		Reject: func(ctx context.Context, tx *sql.Tx) error { return d.Keys.RejectCap(ctx, tx, operation) },
	})
	if err != nil {
		fail(w, http.StatusPaymentRequired, "credit reservation was refused")
		return
	}
	if created {
		if err := d.Wallet.RetainUncertainReservation(r.Context(), key.Account, operation); err != nil {
			fail(w, http.StatusServiceUnavailable, "reservation durability unavailable")
			return
		}
	}
	if err := flow.advance(requestReserve); err != nil {
		fail(w, http.StatusInternalServerError, "request state unavailable")
		return
	}
	if err := flow.advance(requestDispatch); err != nil {
		fail(w, http.StatusInternalServerError, "request state unavailable")
		return
	}
	base := d.GatewayURL
	if d.ResolveGateway != nil {
		base, err = d.ResolveGateway(r.Context())
		if err != nil {
			fail(w, http.StatusServiceUnavailable, "AI Gateway unavailable")
			return
		}
	}
	request := connect.NewRequest(&message)
	request.Header().Set("Authorization", "Bearer "+assertion)
	request.Header().Set(runCallerAssertionHeader, assertion)
	request.Header().Set("X-Vrooli-AI-Source", "vrooli-credits")
	ctx := r.Context()
	response, err := inferenceconnect.NewInferenceServiceClient(d.client(), strings.TrimRight(base, "/")).Run(ctx, request)
	if err != nil {
		var dial *net.OpError
		if errors.As(err, &dial) && dial.Op == "dial" {
			if d.Wallet.ReleaseReservation(ctx, key.Account, operation) == nil {
				_ = d.Keys.SettleCap(ctx, operation, 0)
				_ = flow.advance(requestRelease)
			}
			fail(w, http.StatusBadGateway, "AI Gateway could not be reached")
			return
		}
		_ = flow.advance(requestDefer)
		writePublicRunResult(w, http.StatusAccepted, publicRunResult{Status: "pending"})
		return
	}
	result := response.Msg
	if usage := result.GetUsage(); usage != nil && usage.CreditsCharged != nil && usage.GetAccountingStatus() == "settled" {
		state := "completed"
		if result.GetError() != nil {
			state = "unknown"
		}
		accounting := receipt{OperationID: operation, FeatureID: owner.Feature.ID, State: state, Credits: usage.GetCreditsCharged(), CostMicros: usage.GetCostMicros(), CostKnown: true, PromptTokens: int(usage.GetInputTokens()), CompletionTokens: int(usage.GetOutputTokens()), Model: result.GetModel(), Supplier: result.GetProvider()}
		if err := flow.settle(ctx, key, accounting); err != nil {
			_ = flow.advance(requestDefer)
			writePublicRunResult(w, http.StatusAccepted, publicRunResult{Status: "pending"})
			return
		}
		if result.GetError() != nil {
			writePublicRunResult(w, http.StatusOK, publicRunResult{Status: "settled", ChargedCredits: usage.CreditsCharged})
			return
		}
		writePublicRunResult(w, http.StatusOK, publicRunResult{Status: "settled", ValueJSON: json.RawMessage(result.GetValueJson()), Validated: result.GetValidated(), ChargedCredits: usage.CreditsCharged})
		return
	}
	if result.GetProviderCallsMade() == 0 && result.GetRecovery() == nil {
		if d.Wallet.ReleaseReservation(ctx, key.Account, operation) == nil {
			_ = d.Keys.SettleCap(ctx, operation, 0)
			_ = flow.advance(requestRelease)
		}
		writePublicRunResult(w, http.StatusBadGateway, publicRunResult{Status: "released", Error: "inference was not dispatched"})
		return
	}
	_ = flow.advance(requestDefer)
	writePublicRunResult(w, http.StatusAccepted, publicRunResult{Status: "pending"})
}

func writePublicRunResult(w http.ResponseWriter, status int, result publicRunResult) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func gatewayRunDigest(message *inferencev1.RunRequest) (string, error) {
	clone := proto.Clone(message).(*inferencev1.RunRequest)
	clone.RootOperationId = ""
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(clone)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func (s KeyStore) beginGatewayRun(ctx context.Context, operation string, key Key, featureID, role, digest string) error {
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO ai_gateway_run_operations(operation_id,key_id,feature_id,role,request_digest) VALUES($1,$2,$3,$4,$5) ON CONFLICT(operation_id) DO NOTHING`, operation, key.ID, featureID, role, digest); err != nil {
		return err
	}
	var oldKey, oldFeature, oldRole, oldDigest string
	if err := s.DB.QueryRowContext(ctx, `SELECT key_id,feature_id,role,request_digest FROM ai_gateway_run_operations WHERE operation_id=$1`, operation).Scan(&oldKey, &oldFeature, &oldRole, &oldDigest); err != nil {
		return err
	}
	if oldKey != key.ID || oldFeature != featureID || oldRole != role || oldDigest != digest {
		return fmt.Errorf("Gateway run operation conflicts with prior scope")
	}
	return nil
}

func (s KeyStore) IsGatewayRun(ctx context.Context, operation string, key Key) (bool, error) {
	var found int
	err := s.DB.QueryRowContext(ctx, `SELECT 1 FROM ai_gateway_run_operations WHERE operation_id=$1 AND key_id=$2`, operation, key.ID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
