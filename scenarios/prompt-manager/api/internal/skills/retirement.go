package skills

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"prompt-manager/internal/store"
)

// SkillRetirementReview is the reviewed caller/use closure presented to the
// owner mutation. Counts are required to be zero, and each inventory must be
// named in EvidenceRefs so the decision is auditable. This is an owner
// workflow contract: callers must produce these facts from complete scans;
// graph orphanhood by itself is not sufficient.
type SkillRetirementReview struct {
	Actor                   string   `json:"actor"`
	Reason                  string   `json:"reason"`
	GraphInboundEdges       int      `json:"graphInboundEdges"`
	RepositoryMentions      int      `json:"repositoryMentions"`
	UsageReads90d           int      `json:"usageReads90d"`
	ExternalCallersReviewed bool     `json:"externalCallersReviewed"`
	ExternalCallerCount     int      `json:"externalCallerCount"`
	EvidenceRefs            []string `json:"evidenceRefs"`
}

var errSkillRetirementReview = errors.New("skill retirement review is incomplete or reports callers/use")

type skillRetirementStore interface {
	RetireSkill(ctx context.Context, id, actor, reason string, evidence []string) error
}

// RetireSkill removes a core/local authored skill from active discovery only
// after an explicit zero-caller/use review. The underlying store preserves the
// full skill directory and appends retirement provenance; this method does not
// delete metrics or history.
func (h *Handlers) RetireSkill(ctx context.Context, id string, review SkillRetirementReview) error {
	review.Actor, review.Reason = strings.TrimSpace(review.Actor), strings.TrimSpace(review.Reason)
	if id == "" || review.Actor == "" || review.Reason == "" ||
		review.GraphInboundEdges != 0 || review.RepositoryMentions != 0 ||
		review.UsageReads90d != 0 || !review.ExternalCallersReviewed || review.ExternalCallerCount != 0 ||
		!hasRequiredRetirementEvidence(review.EvidenceRefs) {
		return errSkillRetirementReview
	}
	if h.metrics == nil {
		return fmt.Errorf("cannot retire skill %s without current persisted usage metrics", id)
	}
	metrics, err := h.metricsFor(ctx).Get(id)
	if err != nil {
		return fmt.Errorf("checking persisted skill usage for %s: %w", id, err)
	}
	if metrics == nil || metrics.UsageCount != 0 || metrics.LastUsed != nil {
		return fmt.Errorf("cannot retire skill %s: persisted usage indicates use", id)
	}
	if _, folder, err := h.storeFor(ctx).FindByID(id); err != nil {
		return err
	} else if folder != "core" && folder != "local" {
		return fmt.Errorf("cannot retire skill %s from %s", id, folder)
	}
	retirer, ok := h.storeFor(ctx).(skillRetirementStore)
	if !ok {
		return errors.New("skill store does not support evidence-backed retirement")
	}
	if err := retirer.RetireSkill(ctx, id, review.Actor, review.Reason, review.EvidenceRefs); err != nil {
		return err
	}
	h.invalidateGraph()
	return nil
}

func hasRequiredRetirementEvidence(refs []string) bool {
	required := map[string]bool{"graph": false, "repository": false, "usage": false, "external-callers": false}
	for _, ref := range refs {
		parts := strings.SplitN(strings.TrimSpace(ref), ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			continue
		}
		if _, ok := required[parts[0]]; ok {
			required[parts[0]] = true
		}
	}
	for _, present := range required {
		if !present {
			return false
		}
	}
	return true
}

// RetireSkill delegates the handler lifecycle to the canonical file store.
func (a *StoreAdapter) RetireSkill(ctx context.Context, id, actor, reason string, evidence []string) error {
	retirer, ok := a.skillStore.(interface {
		Retire(context.Context, string, string, string, []string) (*store.SkillRetirement, error)
	})
	if !ok {
		return errors.New("underlying skill store does not support retirement")
	}
	_, err := retirer.Retire(ctx, id, actor, reason, evidence)
	return err
}
