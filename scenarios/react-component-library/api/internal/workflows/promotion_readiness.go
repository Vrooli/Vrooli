package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
)

// NewPromotionReadinessReader composes existing source-of-truth reads. It
// intentionally has no writer: promotion remains an operator-confirmed ingest
// followed by an operator-confirmed apply/reapply.
func NewPromotionReadinessReader(c components.Service, a adoptions.Service) PromotionReadinessReader {
	return promotionReadinessReader{components: c, adoptions: a}
}

type promotionReadinessReader struct {
	components components.Service
	adoptions  adoptions.Service
}

func (r promotionReadinessReader) PromotionReadiness(ctx context.Context, in PromotionReadinessInput) (PromotionReadiness, error) {
	in.AssetID, in.OriginScenario, in.Version = strings.TrimSpace(in.AssetID), strings.TrimSpace(in.OriginScenario), strings.TrimSpace(in.Version)
	if in.AssetID == "" {
		return PromotionReadiness{}, fmt.Errorf("asset_id is required")
	}
	c, err := r.components.Get(ctx, in.AssetID)
	if err != nil {
		return PromotionReadiness{}, err
	}
	version := in.Version
	if version == "" {
		version = c.DraftVersion
	}
	if version == "" {
		version = c.LatestVersion
	}
	if version == "" {
		version = c.Version
	}
	v, err := r.components.GetVersion(ctx, c.ID, version)
	if err != nil {
		return PromotionReadiness{}, err
	}
	stories, err := r.components.ListStories(ctx, components.StoryQuery{ComponentID: c.ID, Version: version, Limit: 20})
	if err != nil {
		return PromotionReadiness{}, err
	}
	availableStoryCount := 0
	for _, story := range stories {
		var contract components.StoryContract
		if err := json.Unmarshal([]byte(story.ContractJSON), &contract); err != nil {
			return PromotionReadiness{}, fmt.Errorf("decode story contract for %s@%s: %w", c.LibraryID, version, err)
		}
		availableStoryCount += len(contract.Stories)
	}
	out := PromotionReadiness{AssetID: c.ID, LibraryID: c.LibraryID, SelectedVersion: version, OriginScenario: in.OriginScenario, RequiredExampleCount: 1, AvailableExampleCount: availableStoryCount, ParityReportPresent: v.ParityReport != nil, NextValidationCommand: "vrooli scenario test react-component-library"}
	for _, d := range c.Dependencies {
		out.DependencyLibraryIDs = append(out.DependencyLibraryIDs, d.LibraryID)
	}
	if v.ParityReport != nil {
		out.OriginFiles = append(out.OriginFiles, v.ParityReport.OriginFiles...)
		out.ParityWaived = v.ParityReport.Acknowledged
		for _, f := range v.ParityReport.Findings {
			out.ParityFindings = append(out.ParityFindings, f.Code+": "+f.Message)
		}
	}
	if !out.ParityReportPresent {
		out.Blockers = append(out.Blockers, "selected version has no origin parity report")
	}
	if out.AvailableExampleCount < out.RequiredExampleCount {
		out.Blockers = append(out.Blockers, "selected version has no required story")
	}
	if in.OriginScenario == "" {
		out.Blockers = append(out.Blockers, "origin_scenario is required to verify replacement and drift")
	} else {
		rows, err := r.adoptions.List(ctx, adoptions.ListQuery{ComponentID: c.ID, Scenario: in.OriginScenario, Limit: 200})
		if err != nil {
			return PromotionReadiness{}, err
		}
		for _, row := range rows {
			if row.AdoptedVersion == version {
				out.OriginReplacementPresent = true
				out.OriginReplacementClean = row.LibraryVersionStatus == adoptions.LibraryVersionStatusCurrent && row.LocalStatus == adoptions.LocalStatusClean
				break
			}
		}
		if !out.OriginReplacementPresent {
			out.Blockers = append(out.Blockers, "origin scenario has no recorded replacement adoption at selected version")
		} else if !out.OriginReplacementClean {
			out.Blockers = append(out.Blockers, "origin replacement drift is not clean; refresh or reapply it")
		}
	}
	out.Ready = len(out.Blockers) == 0
	return out, nil
}
