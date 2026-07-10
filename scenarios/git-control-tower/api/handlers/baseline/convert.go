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
