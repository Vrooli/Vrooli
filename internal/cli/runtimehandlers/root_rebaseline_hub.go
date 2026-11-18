package runtimehandlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/rootrebaselinedecision"
	"github.com/vrooli/api-core/servicetoken"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const rootDecisionAudience = "scenario-authenticator:notification-hub-root-rebaseline"
const rootDecisionScope = "notification-hub:root-rebaseline"

type rootDecisionTerms = rootrebaselinedecision.Terms
type rootDecisionLookup struct {
	AskID          string `json:"ask_id"`
	RequesterRef   string `json:"requester_ref"`
	Digest         string `json:"canonical_terms_sha256"`
	State          string `json:"state"`
	Answer         string `json:"answer"`
	IdentityKind   string `json:"answered_by_identity_kind"`
	IdentitySource string `json:"answered_by_identity_source"`
	Deadline       string `json:"deadline"`
	AnsweredAt     string `json:"answered_at"`
	ConsumedAt     string `json:"consumed_at"`
}

type rootDecisionClient struct {
	base  string
	http  *http.Client
	token func(context.Context) (string, error)
}

type rootDecisionHTTPError struct{ Status int }

func (e rootDecisionHTTPError) Error() string {
	return fmt.Sprintf("notification-hub decision request failed: HTTP %d", e.Status)
}

func newRootDecisionClient(ctx context.Context) (*rootDecisionClient, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
	if err != nil {
		return nil, err
	}
	tokens := servicetoken.New(servicetoken.Config{Name: "root rebaseline", Credential: credentialauthority.ServiceCredential("scenario-authenticator/service-principal/root-rebaseline/notification-hub-root-rebaseline", "credential"), Expect: servicetoken.ExpectIdentity("root-rebaseline", rootDecisionAudience, rootDecisionScope)})
	return &rootDecisionClient{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 10 * time.Second}, token: tokens.Token}, nil
}
func (c *rootDecisionClient) post(ctx context.Context, path string, body any, result any) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rootDecisionHTTPError{Status: resp.StatusCode}
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16384)).Decode(result); err != nil {
		return fmt.Errorf("invalid notification-hub response: %w", err)
	}
	return nil
}
func verifyRootDecision(got rootDecisionLookup, askID string, terms rootDecisionTerms, digest string, now time.Time) (bool, error) {
	if got.AskID != askID || askID == "" || got.RequesterRef != terms.RequesterRef || got.Digest != digest || !sameDecisionDeadline(got.Deadline, terms.Deadline) {
		return false, fmt.Errorf("notification-hub decision identity does not match the qualified transition")
	}
	deadline, err := time.Parse(time.RFC3339, terms.Deadline)
	if err != nil || !now.Before(deadline) {
		return false, fmt.Errorf("notification-hub decision expired")
	}
	switch got.State {
	case "pending", "escalated":
		return false, nil
	case "answered":
	default:
		return false, fmt.Errorf("notification-hub decision closed in state %q", got.State)
	}
	if got.Answer != "approve" {
		return false, fmt.Errorf("notification-hub decision was not approved")
	}
	if got.IdentityKind != "verified_human" || (got.IdentitySource != "cloudflare_access" && got.IdentitySource != "scenario_authenticator_step_up") {
		return false, fmt.Errorf("notification-hub answer lacks verified human identity")
	}
	answered, err := time.Parse(time.RFC3339Nano, got.AnsweredAt)
	if err != nil || answered.After(now) || !answered.Before(deadline) {
		return false, fmt.Errorf("notification-hub answer time is invalid")
	}
	return true, nil
}

// sameDecisionDeadline accepts the same instant written in either RFC3339
// seconds (the terms producer) or RFC3339Nano (the Hub's stored ask). Comparing
// the strings directly refused a perfectly valid decision whenever the Hub
// preserved fractional seconds.
func sameDecisionDeadline(hub, terms string) bool {
	hubTime, hubErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(hub))
	termTime, termErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(terms))
	if hubErr != nil || termErr != nil {
		return strings.TrimSpace(hub) == strings.TrimSpace(terms)
	}
	return hubTime.Equal(termTime)
}

// The local record is a locator, never authority. The Hub consumes each verified
// approval once. Requalification runs under the root transition lock on every
// invocation before this function is called.
func confirmRootViaHub(ctx context.Context, c *rootDecisionClient, plan runtimeapp.RebaselinePlan, bundleSHA, pendingPath string, out io.Writer) (bool, error) {
	if err := rootrebaselinedecision.ValidateChanges(plan.Changes); err != nil {
		return false, fmt.Errorf("cannot present all changes in Notification Hub: %w; use make rebaseline from an operator terminal", err)
	}
	now := time.Now()
	operation := "contract_change"
	if plan.Continuation {
		operation = "continuation"
	}
	terms, err := loadPreparedHubDecisionLocator(pendingPath)
	if err != nil {
		return false, fmt.Errorf("read pending rebaseline decision: %w", err)
	}
	deadline, _ := time.Parse(time.RFC3339, terms.Deadline)
	if terms.RequesterRef != "" && (terms.Version != 2 || terms.Operation != operation || deadline.IsZero() || terms.CandidateSHA != plan.CandidateSHA || terms.ServingSHA != plan.ServingSHA || terms.WatchdogSHA != plan.WatchdogSHA || terms.BundleSHA != bundleSHA || terms.CandidateContract != plan.CandidateContract || terms.ServingContract != plan.ServingContract || !reflect.DeepEqual(terms.Changes, plan.Changes)) {
		return false, fmt.Errorf("the prepared bundle's decision no longer matches; start a new `make rebaseline-hub` and retain this snapshot for recovery")
	}
	if terms.RequesterRef != "" {
		priorDigest, err := rootrebaselinedecision.Digest(terms)
		if err != nil {
			return false, err
		}
		var prior rootDecisionLookup
		err = c.post(ctx, "/api/v1/internal/root-rebaseline/lookup", map[string]string{"requester_ref": terms.RequesterRef, "canonical_terms_sha256": priorDigest}, &prior)
		var httpErr rootDecisionHTTPError
		if err != nil && !(errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound) {
			return false, err
		}
		if err == nil {
			if prior.AskID == "" || prior.RequesterRef != terms.RequesterRef || prior.Digest != priorDigest || !sameDecisionDeadline(prior.Deadline, terms.Deadline) {
				return false, fmt.Errorf("notification-hub decision identity does not match the qualified transition")
			}
			if prior.ConsumedAt != "" {
				if err := verifyConsumedRootDecision(prior, prior.AskID, terms, priorDigest); err != nil {
					return false, err
				}
				terms = rootDecisionTerms{} // consumed authority cannot authorize a second transition
			}
		}
		if !now.Before(deadline) {
			terms = rootDecisionTerms{}
		}
	}
	if terms.RequesterRef == "" {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return false, err
		}
		terms = rootDecisionTerms{Version: 2, Operation: operation, RequesterRef: "root-rebaseline-" + hex.EncodeToString(nonce[:]), Deadline: now.UTC().Add(20 * time.Minute).Truncate(time.Second).Format(time.RFC3339), CandidateSHA: plan.CandidateSHA, ServingSHA: plan.ServingSHA, WatchdogSHA: plan.WatchdogSHA, BundleSHA: bundleSHA, CandidateContract: plan.CandidateContract, ServingContract: plan.ServingContract, Changes: plan.Changes}
		if err := savePreparedHubDecisionLocator(pendingPath, terms); err != nil {
			return false, fmt.Errorf("save pending rebaseline decision: %w", err)
		}
	}
	encoded, err := json.Marshal(terms)
	if err != nil {
		return false, err
	}
	digest, err := rootrebaselinedecision.Digest(terms)
	if err != nil {
		return false, err
	}
	var request map[string]any
	if err := json.Unmarshal(encoded, &request); err != nil {
		return false, err
	}
	request["canonical_terms_sha256"] = digest
	var published struct {
		AskID        string `json:"ask_id"`
		RequesterRef string `json:"requester_ref"`
	}
	if err := c.post(ctx, "/api/v1/internal/root-rebaseline/publish", request, &published); err != nil {
		return false, err
	}
	if published.AskID == "" || published.RequesterRef != terms.RequesterRef {
		return false, fmt.Errorf("notification-hub returned a mismatched decision")
	}
	lookup := map[string]string{"requester_ref": terms.RequesterRef, "canonical_terms_sha256": digest}
	var got rootDecisionLookup
	if err := c.post(ctx, "/api/v1/internal/root-rebaseline/lookup", lookup, &got); err != nil {
		return false, err
	}
	yes, err := verifyRootDecision(got, published.AskID, terms, digest, time.Now())
	if err != nil {
		return false, err
	}
	if !yes {
		fmt.Fprintf(out, "Review the rebaseline decision in Notification Hub, then run HUB_SNAPSHOT='%s' make rebaseline-hub-resume.\n", strings.ReplaceAll(filepath.Dir(pendingPath), "'", "'\\''"))
		return false, fmt.Errorf("rebaseline decision is pending; no transition was made")
	}
	// The Hub commits consumption atomically with verifying the answer. A lost
	// acknowledgement cannot authorize effects; retrying it is refused by the Hub.
	var consumed rootDecisionLookup
	if err := c.post(ctx, "/api/v1/internal/root-rebaseline/consume", lookup, &consumed); err != nil {
		var httpErr rootDecisionHTTPError
		if errors.As(err, &httpErr) && httpErr.Status == http.StatusConflict {
			return false, fmt.Errorf("decision could not be consumed; rerun HUB_SNAPSHOT='%s' make rebaseline-hub-resume to request a fresh decision: %w", strings.ReplaceAll(filepath.Dir(pendingPath), "'", "'\\''"), err)
		}
		return false, err
	}
	verified, err := verifyRootDecision(consumed, published.AskID, terms, digest, time.Now())
	if err != nil || !verified {
		return false, err
	}
	consumedAt, err := time.Parse(time.RFC3339Nano, consumed.ConsumedAt)
	answeredAt, answerErr := time.Parse(time.RFC3339Nano, consumed.AnsweredAt)
	if err != nil || answerErr != nil || consumedAt.Before(answeredAt) || consumedAt.After(time.Now()) {
		return false, fmt.Errorf("notification-hub did not confirm decision consumption")
	}
	return true, nil
}

// verifyConsumedHubDecision re-reads authority for recovery even when the root
// already serves; the local locator alone cannot authorize companions.
func verifyConsumedHubDecision(ctx context.Context, c *rootDecisionClient, pendingPath, bundleSHA, candidateSHA, watchdogSHA string) error {
	terms, err := loadPreparedHubDecisionLocator(pendingPath)
	if err != nil || terms.RequesterRef == "" || terms.BundleSHA != bundleSHA || terms.CandidateSHA != candidateSHA || terms.WatchdogSHA != watchdogSHA {
		return fmt.Errorf("no matching approved Hub decision for prepared bundle")
	}
	digest, err := rootrebaselinedecision.Digest(terms)
	if err != nil {
		return err
	}
	var got rootDecisionLookup
	if err := c.post(ctx, "/api/v1/internal/root-rebaseline/lookup", map[string]string{"requester_ref": terms.RequesterRef, "canonical_terms_sha256": digest}, &got); err != nil {
		return err
	}
	return verifyConsumedRootDecision(got, got.AskID, terms, digest)
}

func verifyConsumedRootDecision(got rootDecisionLookup, askID string, terms rootDecisionTerms, digest string) error {
	consumed, err := time.Parse(time.RFC3339Nano, got.ConsumedAt)
	answered, answerErr := time.Parse(time.RFC3339Nano, got.AnsweredAt)
	if err != nil || answerErr != nil || consumed.Before(answered) || consumed.After(time.Now()) {
		return fmt.Errorf("Hub decision was not consumed")
	}
	verified, err := verifyRootDecision(got, askID, terms, digest, consumed)
	if err != nil {
		return fmt.Errorf("Hub decision is not an approved human decision: %w", err)
	}
	if !verified {
		return fmt.Errorf("Hub decision remains pending")
	}
	return nil
}
