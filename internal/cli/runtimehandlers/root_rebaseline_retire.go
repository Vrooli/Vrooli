package runtimehandlers

import (
	"context"
	"fmt"

	"github.com/vrooli/api-core/rootrebaselinedecision"
)

// retireSnapshotDecision seals an expired, unused exact decision at the Hub.
// Missing or invalid locators, unreachable Hub, and consumed decisions all fail
// closed. The caller must hold the snapshot lease until deletion completes.
func retireSnapshotDecision(ctx context.Context, c *rootDecisionClient, snapshot string) error {
	identity, err := loadPreparedSnapshotDecision(snapshot)
	if err != nil {
		return err
	}
	terms := identity.terms
	digest, err := rootrebaselinedecision.Digest(terms)
	if err != nil {
		return err
	}
	var result struct {
		RequesterRef string `json:"requester_ref"`
		Digest       string `json:"canonical_terms_sha256"`
		State        string `json:"state"`
	}
	if err := c.post(ctx, "/api/v1/internal/root-rebaseline/retire", map[string]string{"requester_ref": terms.RequesterRef, "canonical_terms_sha256": digest}, &result); err != nil {
		return err
	}
	if result.RequesterRef != terms.RequesterRef || result.Digest != digest || result.State != "retired" {
		return fmt.Errorf("Hub retirement identity mismatch")
	}
	return nil
}
