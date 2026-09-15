package apply

import (
	"brand-manager/internal/apply"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply"
)

// resultToProto converts the domain Result into the wire ApplyResponse.
func resultToProto(r apply.Result) *applyv1.ApplyResponse {
	applied := make([]*applyv1.ApplyAction, 0, len(r.Applied))
	for _, a := range r.Applied {
		applied = append(applied, &applyv1.ApplyAction{
			Type:    a.Type,
			File:    a.File,
			Element: a.Element,
		})
	}
	skipped := make([]*applyv1.SkipReason, 0, len(r.Skipped))
	for _, s := range r.Skipped {
		skipped = append(skipped, &applyv1.SkipReason{
			Element: s.Element,
			Reason:  s.Reason,
		})
	}
	return &applyv1.ApplyResponse{
		Scenario:     r.Scenario,
		BrandId:      r.BrandID,
		BrandVersion: int32(r.BrandVersion),
		DryRun:       r.DryRun,
		Applied:      applied,
		Skipped:      skipped,
	}
}
