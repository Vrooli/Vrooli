package evidence

import (
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/certification"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SummarySchemaVersion is the wire schema of ReleaseEvidence.
const SummarySchemaVersion = "1"

// Cell is the evaluated disposition of one profile cell with the evidence
// that decided it.
type Cell struct {
	CaseID        string             `json:"case_id"`
	Lane          certification.Lane `json:"lane"`
	Disposition   Disposition        `json:"disposition"`
	Required      bool               `json:"required"`
	RecordID      string             `json:"record_id,omitempty"`
	OperationID   string             `json:"operation_id,omitempty"`
	ObservationID string             `json:"observation_id,omitempty"`
	ReceiptRefs   []string           `json:"receipt_refs,omitempty"`
	ObservedAt    time.Time          `json:"observed_at,omitempty"`
	Reason        string             `json:"reason,omitempty"`
}

// Summary is the evaluation of one release on one target against a profile.
type Summary struct {
	SchemaVersion string    `json:"schema_version"`
	ProfileID     string    `json:"profile_id"`
	ReleaseDigest string    `json:"release_digest"`
	TargetKey     string    `json:"target_key"`
	Cells         []Cell    `json:"cells"`
	RequiredCells int       `json:"required_cells"`
	Passed        bool      `json:"passed"`
	BlockingCells []string  `json:"blocking_cells,omitempty"`
	ProducerRef   string    `json:"producer_ref"`
	EvaluatedAt   time.Time `json:"evaluated_at"`
}

// Evaluate decides every cell of the profile for one release and target.
//
// Precedence per cell, considering only records bound to the exact release
// digest and target key (records for another release or target are
// incompatible and count as absent; the cell reason names them):
//  1. a failed record that no later record withdraws -> failed
//  2. otherwise the latest passed record -> passed
//  3. otherwise the latest skipped/unsupported/unavailable record -> that word
//  4. otherwise -> missing
//
// A failed record is never masked by a later pass; withdrawal is an explicit
// record that names the failed id and carries a reason.
func Evaluate(profile *Profile, records []Record, releaseDigest, targetKey string, now time.Time) Summary {
	releaseDigest = strings.ToLower(strings.TrimSpace(releaseDigest))
	out := Summary{SchemaVersion: SummarySchemaVersion, ProfileID: profile.ID, ReleaseDigest: releaseDigest, TargetKey: targetKey, ProducerRef: ProducerRef, EvaluatedAt: now.UTC(), RequiredCells: len(profile.Cells)}
	sorted := append([]Record(nil), records...)
	SortRecords(sorted)
	withdrawn := map[string]bool{}
	for _, r := range sorted {
		if r.Withdraws != "" && strings.EqualFold(r.Binding.ReleaseDigest, releaseDigest) && r.Binding.TargetKey == targetKey {
			withdrawn[r.Withdraws] = true
		}
	}
	byCell := map[CellID][]Record{}
	foreign := map[CellID][]Record{}
	for _, r := range sorted {
		id := CellID{CaseID: r.CaseID, Lane: r.Lane}
		if !strings.EqualFold(r.Binding.ReleaseDigest, releaseDigest) || (targetKey != "" && r.Binding.TargetKey != targetKey) {
			foreign[id] = append(foreign[id], r)
			continue
		}
		byCell[id] = append(byCell[id], r)
	}
	for _, id := range profile.Cells {
		cell := Cell{CaseID: id.CaseID, Lane: id.Lane, Required: true, Disposition: DispositionMissing}
		var failed, passed, other *Record
		for i := range byCell[id] {
			r := &byCell[id][i]
			switch r.Disposition {
			case DispositionFailed:
				if !withdrawn[r.ID] {
					failed = r
				}
			case DispositionPassed:
				passed = r
			default:
				other = r
			}
		}
		switch {
		case failed != nil:
			fill(&cell, failed, "failed record for candidate is in force")
		case passed != nil:
			fill(&cell, passed, "")
		case other != nil:
			fill(&cell, other, other.Reason)
		default:
			cell.Reason = "no evidence record for this release and target"
			if f := foreign[id]; len(f) > 0 {
				last := f[len(f)-1]
				cell.Reason = fmt.Sprintf("evidence exists only for release %s on target %s (incompatible)", short(last.Binding.ReleaseDigest), last.Binding.TargetKey)
			}
		}
		if cell.Disposition != DispositionPassed {
			out.BlockingCells = append(out.BlockingCells, id.String())
		}
		out.Cells = append(out.Cells, cell)
	}
	out.Passed = releaseDigest != "" && len(out.BlockingCells) == 0 && out.RequiredCells > 0
	return out
}

func fill(cell *Cell, r *Record, reason string) {
	cell.Disposition = r.Disposition
	cell.RecordID = r.ID
	cell.OperationID = r.Binding.OperationID
	cell.ObservationID = r.Binding.ObservationID
	cell.ReceiptRefs = append([]string(nil), r.ReceiptRefs...)
	cell.ObservedAt = r.ObservedAt
	if reason == "" {
		reason = r.Reason
	}
	cell.Reason = reason
}

func short(digest string) string {
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

// Proto projects the summary onto the wire.
func (s Summary) Proto() *evidencev1.ReleaseEvidence {
	out := &evidencev1.ReleaseEvidence{
		SchemaVersion: s.SchemaVersion, ProfileId: s.ProfileID, ReleaseDigest: s.ReleaseDigest, TargetKey: s.TargetKey,
		RequiredCells: int32(s.RequiredCells), Passed: s.Passed, BlockingCells: append([]string(nil), s.BlockingCells...),
		ProducerRef: s.ProducerRef, EvaluatedAt: timestamppb.New(s.EvaluatedAt),
	}
	for _, c := range s.Cells {
		cell := &evidencev1.EvidenceCell{
			CaseId: c.CaseID, Lane: string(c.Lane), Disposition: c.Disposition.Proto(), RecordId: c.RecordID,
			OperationId: c.OperationID, ObservationId: c.ObservationID, ReceiptRefs: append([]string(nil), c.ReceiptRefs...),
			Reason: c.Reason, Required: c.Required,
		}
		if !c.ObservedAt.IsZero() {
			cell.ObservedAt = timestamppb.New(c.ObservedAt)
		}
		out.Cells = append(out.Cells, cell)
	}
	return out
}

// CoverageVerdict projects the summary onto the Deployment Manager
// reference-only target verdict. The target platform carries the cloud
// target key so DM's ramp-evidence coverage matches it against the review's
// target set exactly; refs point at every cell's record, never at bytes.
// Disposition is PASSED only when every required cell passed.
func (s Summary) CoverageVerdict(runID string, now time.Time) (*commonv1.TargetVerdict, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("coverage verdict requires a run id")
	}
	if strings.TrimSpace(s.TargetKey) == "" {
		return nil, fmt.Errorf("coverage verdict requires a target key")
	}
	disposition := commonv1.Disposition_DISPOSITION_FAILED
	if s.Passed {
		disposition = commonv1.Disposition_DISPOSITION_PASSED
	}
	verdict := &commonv1.TargetVerdict{
		Target:        &commonv1.EvidenceTarget{Ramp: ProducerRef, Platform: s.TargetKey, Os: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST},
		Disposition:   disposition,
		RunId:         runID,
		Detail:        fmt.Sprintf("%s release %s: %d required cells, blocking %s", s.ProfileID, short(s.ReleaseDigest), s.RequiredCells, strings.Join(s.BlockingCells, ",")),
		EvidenceClass: "cloud-capability-profile",
	}
	for _, c := range s.Cells {
		if c.RecordID == "" {
			continue
		}
		verdict.Refs = append(verdict.Refs, &commonv1.EvidenceRef{
			Producer: ProducerRef, ArtifactId: c.RecordID, Kind: "evidence-record:" + c.CaseID + "/" + string(c.Lane),
			Checksum: "sha256:" + strings.TrimPrefix(s.ReleaseDigest, "sha256:"), CreatedAt: timestamppb.New(now.UTC()),
		})
	}
	return verdict, nil
}
