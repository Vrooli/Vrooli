package baseline

import (
	"time"

	bl "git-control-tower/internal/baseline"
	"git-control-tower/internal/git"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// rfc3339 formats a time as RFC3339, returning "" for the zero value.
func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func gitToProto(st git.State) *baselinesv1.GitState {
	return &baselinesv1.GitState{
		Sha:           st.Sha,
		Branch:        st.Branch,
		Detached:      st.Detached,
		Dirty:         st.Dirty,
		DirtySummary:  st.DirtySummary,
		CommitMessage: st.CommitMessage,
		CommitAuthor:  st.CommitAuthor,
		CommitDate:    rfc3339(st.CommitDate),
		Sandboxed:     st.Sandboxed,
	}
}

func manifestToProto(m bl.BaselineManifest) *baselinesv1.BaselineManifest {
	out := &baselinesv1.BaselineManifest{
		Name:      m.Name,
		Scenario:  m.Scenario,
		Branch:    m.Branch,
		CreatedAt: rfc3339(m.CreatedAt),
		CreatedBy: m.CreatedBy,
		Git:       gitToProto(m.Git),
		Run: &baselinesv1.RunAnchor{
			RunId:                           m.Run.RunID,
			CapturedAt:                      rfc3339(m.Run.CapturedAt),
			CaptureProfile:                  m.Run.CaptureProfile,
			TreeDigest:                      m.Run.TreeDigest,
			PhaseSetDigest:                  m.Run.PhaseSetDigest,
			DescriptorSnapshotRef:           m.Run.DescriptorSnapshotRef,
			DescriptorSnapshotDigest:        m.Run.DescriptorSnapshotDigest,
			DescriptorSnapshotSchemaVersion: int32(m.Run.DescriptorSnapshotSchemaVersion),
		},
		SchemaVersion: int32(m.SchemaVersion),
	}
	if m.Migration != nil {
		out.Migration = &baselinesv1.MigrationInfo{
			FromSchemaVersion: int32(m.Migration.FromSchemaVersion),
			MigratedAt:        rfc3339(m.Migration.MigratedAt),
			DegradedReasons:   append([]string(nil), m.Migration.DegradedReasons...),
		}
	}
	return out
}

func collectionToProto(collection bl.CollectionManifest) *baselinesv1.BaselineCollection {
	coverage := collection.Coverage()
	out := &baselinesv1.BaselineCollection{
		Name:          collection.Name,
		Branch:        collection.Branch,
		CreatedAt:     rfc3339(collection.CreatedAt),
		UpdatedAt:     rfc3339(collection.UpdatedAt),
		SchemaVersion: int32(collection.SchemaVersion),
		Coverage: &baselinesv1.CollectionCoverage{
			Required: int32(coverage.Required), Ready: int32(coverage.Ready), Pending: int32(coverage.Pending),
			Failed: int32(coverage.Failed), Skipped: int32(coverage.Skipped), Stale: int32(coverage.Stale), Complete: coverage.Complete(),
		},
	}
	for _, member := range collection.Members {
		out.Members = append(out.Members, &baselinesv1.CollectionMember{
			Scenario: member.Scenario, BaselineName: member.BaselineName, Required: member.Required,
			Status: string(member.Status), RunId: member.RunID, GitSha: member.GitSHA, Error: member.Error, UpdatedAt: rfc3339(member.UpdatedAt),
		})
	}
	for _, name := range collection.PathSnapshots {
		out.PathSnapshots = append(out.PathSnapshots, &baselinesv1.PathSnapshotReference{Name: name, Branch: collection.Branch})
	}
	return out
}

func pathEntryToProto(entry bl.PathEntry) *baselinesv1.PathEntry {
	return &baselinesv1.PathEntry{
		Path: entry.Path, Mode: uint32(entry.Mode), Type: entry.Type, Size: entry.Size,
		Digest: entry.Digest, State: string(entry.State), Detail: entry.Detail,
	}
}

func pathSnapshotToProto(snapshot bl.PathSnapshot) *baselinesv1.PathSnapshot {
	out := &baselinesv1.PathSnapshot{
		Name: snapshot.Name, Branch: snapshot.Branch, CreatedAt: rfc3339(snapshot.CreatedAt),
		ExpiresAt:     rfc3339(snapshot.ExpiresAt),
		SchemaVersion: int32(snapshot.SchemaVersion), Selections: append([]string(nil), snapshot.Selections...),
		Classification: "informational-source-evidence", IncludeIgnored: snapshot.Policy.IncludeIgnored, RetainContent: snapshot.Policy.RetainContent, PolicyVersion: int32(snapshot.PolicyVersion),
	}
	for _, entry := range snapshot.Entries {
		out.Entries = append(out.Entries, pathEntryToProto(entry))
	}
	return out
}

func pathSnapshotEstimateToProto(estimate bl.PathSnapshotEstimate) *baselinesv1.PathSnapshotEstimate {
	out := &baselinesv1.PathSnapshotEstimate{
		Selections: append([]string(nil), estimate.Selections...), IncludeIgnored: estimate.Policy.IncludeIgnored, RetainContent: estimate.Policy.RetainContent,
		EligibleFiles: int32(estimate.EligibleFiles), EligibleBytes: estimate.EligibleBytes, ExcludedIgnoredFiles: int32(estimate.ExcludedIgnoredFiles), ExcludedIgnoredBytes: estimate.ExcludedIgnoredBytes,
		ExcludedSensitiveFiles: int32(estimate.ExcludedSensitiveFiles), ExcludedBinaryFiles: int32(estimate.ExcludedBinaryFiles), OversizedFiles: int32(estimate.OversizedFiles),
		RetainedContentBytes: estimate.RetainedContentBytes, RepairRequired: estimate.RequiresRepair(), PolicyVersion: int32(estimate.PolicyVersion),
	}
	for _, contributor := range estimate.TopContributors {
		out.TopContributors = append(out.TopContributors, &baselinesv1.PathSnapshotContributor{Path: contributor.Path, Files: int32(contributor.Files), Bytes: contributor.Bytes})
	}
	for _, issue := range estimate.Issues {
		out.Issues = append(out.Issues, &baselinesv1.PathSnapshotIssue{Code: issue.Code, Severity: issue.Severity, Detail: issue.Detail})
	}
	for _, recommendation := range estimate.Recommendations {
		out.Recommendations = append(out.Recommendations, &baselinesv1.PathSnapshotRecommendation{Selection: recommendation.Selection, Reason: recommendation.Reason})
	}
	return out
}

func sourceDeltaToProto(delta bl.SourceDelta) *baselinesv1.SourceDelta {
	out := &baselinesv1.SourceDelta{Path: delta.Path, Status: delta.Status}
	if delta.Before != nil {
		out.Before = pathEntryToProto(*delta.Before)
	}
	if delta.After != nil {
		out.After = pathEntryToProto(*delta.After)
	}
	return out
}

func collectionDiffMemberToProto(member bl.CollectionDiffMember) *baselinesv1.CollectionDiffMember {
	return &baselinesv1.CollectionDiffMember{
		Scenario: member.Scenario, Required: member.Required, Status: member.Status,
		RunId: member.RunID, Verdict: string(member.Verdict), Detail: member.Detail,
	}
}

// diffResultToProto renders a computed diff into the wire DiffResult message.
func diffResultToProto(res bl.DiffResult) *baselinesv1.DiffResult {
	out := &baselinesv1.DiffResult{
		Baseline:   manifestToProto(res.Manifest),
		CurrentGit: gitToProto(res.CurrentGit),
		Staleness: &baselinesv1.Staleness{
			CommitsSince: int32(res.Staleness.CommitsSince),
			FilesChanged: int32(res.Staleness.FilesChanged),
			LikelyStale:  res.Staleness.LikelyStale,
		},
		Verdict:      string(res.Verdict),
		DirtyWarning: res.DirtyWarning,
		Evidence:     evidenceToProto(res.Evidence),
	}
	out.Phases = append(out.Phases, res.Phases...)
	return out
}

func artifactCatalogToProto(c bl.ArtifactCatalog) *baselinesv1.RunArtifactCatalog {
	return &baselinesv1.RunArtifactCatalog{
		RunId:            c.RunID,
		SchemaVersion:    int32(c.SchemaVersion),
		Digest:           c.Digest,
		Artifacts:        c.Artifacts,
		LegacyDiscovered: c.LegacyDiscovered,
		DegradedReasons:  c.DegradedReasons,
	}
}

func evidenceToProto(e bl.EvidenceComparison) *baselinesv1.EvidenceComparison {
	out := &baselinesv1.EvidenceComparison{
		BaseRunId:       e.BaseRunID,
		CurrentRunId:    e.CurrentRunID,
		BaseCatalog:     artifactCatalogToProto(e.BaseCatalog),
		CurrentCatalog:  artifactCatalogToProto(e.CurrentCatalog),
		DegradedReasons: e.DegradedReasons,
	}
	for _, d := range e.VisualDeltas {
		out.VisualDeltas = append(out.VisualDeltas, &baselinesv1.VisualDelta{
			Page: d.Page, Label: d.Label, Status: d.Status, ChangedFraction: d.ChangedFraction,
		})
	}
	return out
}
