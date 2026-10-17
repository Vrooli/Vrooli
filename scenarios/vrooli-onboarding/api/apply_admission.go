package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/platform-go"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

const applyReviewTTL = 15 * time.Minute

type applyReviewRecord struct {
	Target                   string `json:"target"`
	PlanID                   string `json:"plan_id"`
	PlanDigest               string `json:"plan_digest"`
	Revision                 string `json:"revision"`
	ProfileConsequenceDigest string `json:"profile_consequence_digest,omitempty"`
	ConsentReceiptID         string `json:"consent_receipt_id"`
	ExpiresAt                string `json:"expires_at"`
	ActorSource              string `json:"actor_source"`
	ActorSubject             string `json:"actor_subject"`
	ConsumedBy               string `json:"consumed_by,omitempty"`
	ConsumedOperationID      string `json:"consumed_operation_id,omitempty"`
}

type applyAdmissionRecord struct {
	Target           string `json:"target"`
	PlanID           string `json:"plan_id"`
	PlanDigest       string `json:"plan_digest"`
	ExpectedRevision string `json:"expected_revision"`
	IdempotencyKey   string `json:"idempotency_key"`
	RunID            string `json:"run_id"`
	CreatedAt        string `json:"created_at,omitempty"`
	ActorSource      string `json:"actor_source"`
	ActorSubject     string `json:"actor_subject"`
}

var applyAdmissionMu sync.Mutex

func newOpaqueID(prefix string) (string, error) {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate %s identity: %w", prefix, err)
	}
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func applyReviewPath(receiptID string) (string, error) {
	statePath, err := operatorStatePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(statePath), "apply-reviews", receiptID+".json"), nil
}

func applyAdmissionPath(idempotencyKey string) (string, error) {
	statePath, err := operatorStatePath()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(idempotencyKey))
	return filepath.Join(filepath.Dir(statePath), "apply-admissions", hex.EncodeToString(digest[:])+".json"), nil
}

func applyTargetLockPath(target string) (string, error) {
	statePath, err := operatorStatePath()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(target)))
	return filepath.Join(filepath.Dir(statePath), "apply-target-"+hex.EncodeToString(digest[:])+".lock"), nil
}

func persistAdmissionFile(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, append(data, '\n'), storage.SecretFilePerm)
}

func loadApplyReview(receiptID string) (applyReviewRecord, error) {
	path, err := applyReviewPath(receiptID)
	if err != nil {
		return applyReviewRecord{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return applyReviewRecord{}, err
	}
	var review applyReviewRecord
	if err := json.Unmarshal(data, &review); err != nil {
		return applyReviewRecord{}, fmt.Errorf("decode apply review: %w", err)
	}
	return review, nil
}

func verifiedApplyActor(ctx context.Context) (string, string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || strings.TrimSpace(principal.Subject) == "" {
		return "", "", errors.New("verified apply actor required")
	}
	return string(principal.Source), strings.TrimSpace(principal.Subject), nil
}

func (s *Server) reviewApply(ctx context.Context, request applydomain.ReviewRequest) (applydomain.Review, error) {
	actorSource, actorSubject, err := verifiedApplyActor(ctx)
	if err != nil {
		return applydomain.Review{}, err
	}
	// The review is a server-produced receipt over the exact plan currently
	// shown. A recommendation or a saved draft never creates this record.
	plan, err := s.buildCurrentApplyPlan(ctx, request.Target)
	if err != nil {
		return applydomain.Review{}, err
	}
	if strings.TrimSpace(request.PlanID) == "" || strings.TrimSpace(request.PlanDigest) == "" || strings.TrimSpace(request.ExpectedRevision) == "" {
		return applydomain.Review{}, errors.New("plan_id, plan_digest, and expected_revision are required")
	}
	if request.PlanID != plan.PlanID || request.PlanDigest != plan.Digest || request.ExpectedRevision != plan.Revision {
		return applydomain.Review{}, errors.New("stale or changed apply plan; refresh review")
	}
	profileDigest, err := profileConsequenceDigest(ctx, plan.Target, actorSource, actorSubject)
	if err != nil {
		return applydomain.Review{}, err
	}
	receiptID, err := newOpaqueID("consent")
	if err != nil {
		return applydomain.Review{}, err
	}
	record := applyReviewRecord{Target: plan.Target, PlanID: plan.PlanID, PlanDigest: plan.Digest, Revision: plan.Revision, ProfileConsequenceDigest: profileDigest, ConsentReceiptID: receiptID, ExpiresAt: operatorStateNow().UTC().Add(applyReviewTTL).Format(time.RFC3339), ActorSource: actorSource, ActorSubject: actorSubject}
	path, err := applyReviewPath(receiptID)
	if err != nil {
		return applydomain.Review{}, err
	}
	if err := persistAdmissionFile(path, record); err != nil {
		return applydomain.Review{}, fmt.Errorf("persist apply review: %w", err)
	}
	return applydomain.Review{Target: record.Target, PlanID: record.PlanID, PlanDigest: record.PlanDigest, Revision: record.Revision, ConsentReceiptID: record.ConsentReceiptID, ExpiresAt: record.ExpiresAt}, nil
}

func (s *Server) admitApply(ctx context.Context, request applydomain.StartRequest) (applydomain.Plan, applyReviewRecord, *applyRun, error) {
	if strings.TrimSpace(request.PlanID) == "" || strings.TrimSpace(request.PlanDigest) == "" || strings.TrimSpace(request.ExpectedRevision) == "" || strings.TrimSpace(request.ConsentReceiptID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("plan_id, plan_digest, expected_revision, consent_receipt_id, and idempotency_key are required")
	}
	actorSource, actorSubject, err := verifiedApplyActor(ctx)
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, err
	}
	// Admission is a check-then-persist transaction. The durable files survive
	// an API restart, while this mutex closes the concurrent same-process gap so
	// two retries cannot both create a run before either admission record exists.
	applyAdmissionMu.Lock()
	defer applyAdmissionMu.Unlock()
	statePath, err := operatorStatePath()
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, err
	}
	release, err := platform.AcquireFileLockContext(ctx, filepath.Join(filepath.Dir(statePath), ".apply-admission.lock"))
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, fmt.Errorf("lock apply admission: %w", err)
	}
	defer release()
	path, err := applyAdmissionPath(request.IdempotencyKey)
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, err
	}
	// Resolve a retry before rebuilding the current plan. The first accepted
	// apply may advance the operator-state revision while the client is
	// waiting for its response; that must not turn a lost-response retry into
	// a false stale-plan rejection.
	if data, readErr := os.ReadFile(path); readErr == nil {
		var admission applyAdmissionRecord
		if err := json.Unmarshal(data, &admission); err != nil {
			return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("idempotency record is corrupt")
		}
		if admission.PlanDigest != request.PlanDigest || (admission.PlanID != "" && admission.PlanID != request.PlanID) || (admission.Target != "" && admission.Target != request.Target) || admission.ActorSource != actorSource || admission.ActorSubject != actorSubject {
			return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("idempotency key was already used for different apply input")
		}
		run, runErr := applyRunSnapshot(admission.RunID)
		if runErr {
			return applydomain.Plan{}, applyReviewRecord{}, &run, nil
		}
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("idempotency record points to a missing apply run")
	}
	plan, err := s.buildCurrentApplyPlan(ctx, request.Target)
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, err
	}
	if request.PlanID != plan.PlanID || request.PlanDigest != plan.Digest || request.ExpectedRevision != plan.Revision {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("stale or changed apply plan; refresh review")
	}
	review, err := loadApplyReview(request.ConsentReceiptID)
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("consent receipt was not found")
	}
	expiresAt, err := time.Parse(time.RFC3339, review.ExpiresAt)
	if err != nil || !expiresAt.After(operatorStateNow().UTC()) {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("consent receipt is expired")
	}
	if review.Target != plan.Target || review.PlanID != plan.PlanID || review.PlanDigest != plan.Digest || review.Revision != plan.Revision {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("consent receipt does not match the current apply plan")
	}
	if review.ActorSource != actorSource || review.ActorSubject != actorSubject {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("consent receipt does not match the current apply actor")
	}
	currentProfileDigest, err := profileConsequenceDigest(ctx, request.Target, actorSource, actorSubject)
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, err
	}
	if review.ProfileConsequenceDigest != currentProfileDigest {
		return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("consent receipt is stale because the onboarding profile consequences changed; review the apply plan again")
	}
	// A retry with the same owner operation identity can finish an interrupted
	// admission. The receipt is consumed before the run is published so a
	// crash cannot let a second operation reuse the same consent. The operation
	// id lets the original retry recover when the later durable write failed.
	if review.ConsumedBy != "" {
		if review.ConsumedOperationID != request.IdempotencyKey {
			return applydomain.Plan{}, applyReviewRecord{}, nil, errors.New("consent receipt does not match the current apply plan")
		}
		if run, ok := applyRunSnapshot(review.ConsumedBy); ok {
			return applydomain.Plan{}, review, &run, nil
		}
	}
	run, err := s.buildApplyRun(ctx, plan)
	if err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, err
	}
	if review.ConsumedBy != "" {
		// The first attempt durably reserved this operation but did not publish
		// its run. Reuse the owner identity so the recovery remains single-shot.
		run.ID = review.ConsumedBy
	} else {
		review.ConsumedBy = run.ID
		review.ConsumedOperationID = request.IdempotencyKey
		reviewPath, err := applyReviewPath(review.ConsentReceiptID)
		if err != nil {
			return applydomain.Plan{}, applyReviewRecord{}, nil, err
		}
		if err := persistAdmissionFile(reviewPath, review); err != nil {
			return applydomain.Plan{}, applyReviewRecord{}, nil, fmt.Errorf("persist consent consumption: %w", err)
		}
	}
	// Do not publish an admission record until the exact run it names is
	// durable. The receipt reservation above makes a retry recover the same
	// operation even if this write fails after the reservation committed.
	if err := persistApplyRun(run); err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, fmt.Errorf("persist apply run: %w", err)
	}
	if err := persistAdmissionFile(path, applyAdmissionRecord{Target: plan.Target, PlanID: plan.PlanID, PlanDigest: plan.Digest, ExpectedRevision: plan.Revision, IdempotencyKey: request.IdempotencyKey, RunID: run.ID, CreatedAt: operatorStateNow().UTC().Format(time.RFC3339), ActorSource: actorSource, ActorSubject: actorSubject}); err != nil {
		return applydomain.Plan{}, applyReviewRecord{}, nil, fmt.Errorf("persist apply admission: %w", err)
	}
	return plan, review, &run, nil
}

func profileConsequenceDigest(ctx context.Context, target, actorSource, actorSubject string) (string, error) {
	profile, err := operatorStateService().ProfileSession(ctx)
	if err != nil {
		return "", err
	}
	if profile == nil {
		return "", nil
	}
	if profile.Target != strings.TrimSpace(target) || profile.Actor != strings.TrimSpace(actorSource)+":"+strings.TrimSpace(actorSubject) {
		return "", nil
	}
	return profile.ConsequenceDigest, nil
}
