package main

import (
	"context"

	"git-control-tower/internal/provenance"
)

// enrichBlame joins historical Workspace Sandbox records and the bounded Agent
// Manager report. It deliberately keeps unavailable and private states visible
// without copying private payloads or transcripts.
func (s *Server) enrichBlame(ctx context.Context, repoDir string, result *BlameResponse) {
	result.Warnings = nil
	if s.sandbox == nil {
		result.Warnings = append(result.Warnings, "workspace-sandbox evidence unavailable: client is not configured")
		return
	}
	groups, err := s.sandbox.GetProvenanceByRun(ctx, repoDir)
	if err != nil {
		result.Warnings = append(result.Warnings, "workspace-sandbox evidence unavailable: "+err.Error())
		return
	}
	resultEvidence := make(map[string][]provenance.Evidence)
	runReferences := make(map[string][]provenance.WorkRef)
	for _, group := range groups.RunGroups {
		bundle := ProvenanceChangeBundle{RunID: group.RunID, SandboxID: group.SandboxID, RunOutcome: group.RunOutcome, ConversationID: group.ConversationID, CostUSD: group.CostUSD}
		if group.RunID != "" && s.agentManagerClient != nil {
			report, reportErr := s.agentManagerClient.GetRunReport(ctx, group.RunID)
			if reportErr != nil {
				result.Warnings = append(result.Warnings, "agent-manager run context unavailable for "+group.RunID+": "+reportErr.Error())
			} else {
				for _, ref := range report.WorkReferences {
					if ref.Visibility == "private" || ref.State == "private" || ref.State == "expired" {
						continue
					}
					runReferences[group.RunID] = append(runReferences[group.RunID], provenance.WorkRef{
						Kind: ref.Kind, ID: ref.ID, Revision: ref.Revision, Relationship: ref.Relationship,
						Verified: ref.Verified, Visibility: ref.Visibility, State: ref.State,
						UnavailableReason: ref.UnavailableReason,
					})
				}
				bundle.WorkReferences = append(bundle.WorkReferences, runReferences[group.RunID]...)
				if report.ReceiptsAvailability.State != "available" {
					result.Warnings = append(result.Warnings, "receipts "+report.ReceiptsAvailability.State+" for "+group.RunID+": "+report.ReceiptsAvailability.Reason)
				}
			}
		}
		for _, evidenceFile := range group.Files {
			path := evidenceFile.RelativePath
			if path == "" {
				path = evidenceFile.FilePath
			}
			visibility := evidenceFile.Visibility
			if visibility == "" {
				visibility = "public"
			}
			runOutcome := evidenceFile.RunOutcome
			if runOutcome == "" {
				runOutcome = group.RunOutcome
			}
			resultEvidence[path] = append(resultEvidence[path], provenance.Evidence{
				RunID: group.RunID, SandboxID: group.SandboxID, ContentDigest: evidenceFile.ContentDigest,
				CommitID: evidenceFile.CommitHash, Visibility: visibility, CommitState: evidenceFile.State,
				RunOutcome: runOutcome, ConversationID: evidenceFile.ConversationID,
				CostUSD: evidenceFile.CostUSD, CommittedAt: evidenceFile.CommittedAt,
				Unavailable: evidenceFile.Unavailable, WorkReferences: runReferences[group.RunID],
			})
			bundle.Files = append(bundle.Files, path)
			if len(evidenceFile.Unavailable) > 0 {
				bundle.Gaps = append(bundle.Gaps, evidenceFile.Unavailable...)
			}
		}
		result.Bundles = append(result.Bundles, bundle)
	}
	for i := range result.Files {
		result.Files[i] = enrichedBlameFile(result.Files[i], resultEvidence[result.Files[i].Path])
	}
}
