package capacity

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const (
	PlacementEvidenceQualified  = "qualified"
	PlacementEvidenceFailed     = "failed"
	PlacementEvidenceStale      = "stale"
	PlacementEvidenceMismatched = "mismatched"
	PlacementEvidenceMissing    = "missing"
)

// PlacementTableRow is a reportable role/rung measurement joined to its
// current capacity claim. Measured device/VRAM and current claim accounting are
// deliberately separate facts.
type PlacementTableRow struct {
	ReceiptID                string        `json:"receipt_id,omitempty"`
	ClaimID                  string        `json:"claim_id"`
	ClaimGeneration          int64         `json:"claim_generation"`
	Role                     string        `json:"role"`
	Model                    ModelIdentity `json:"model"`
	Rung                     string        `json:"rung"`
	Device                   string        `json:"device"`
	Suite                    string        `json:"suite,omitempty"`
	SuiteDigest              string        `json:"suite_digest,omitempty"`
	RunID                    string        `json:"run_id,omitempty"`
	ArtifactURI              string        `json:"artifact_uri,omitempty"`
	ArtifactSHA              string        `json:"artifact_sha256,omitempty"`
	Cases                    int           `json:"cases,omitempty"`
	Passed                   bool          `json:"passed"`
	P95LatencyMS             float64       `json:"p95_latency_ms,omitempty"`
	MaxLatencyMS             float64       `json:"max_latency_ms,omitempty"`
	MeasuredVRAMBytes        int64         `json:"measured_vram_bytes"`
	SampledAt                time.Time     `json:"sampled_at,omitempty"`
	EvidenceStatus           string        `json:"evidence_status"`
	Reason                   string        `json:"reason,omitempty"`
	CurrentClaimGeneration   int64         `json:"current_claim_generation,omitempty"`
	CurrentClaimAmountBytes  int64         `json:"current_claim_amount_bytes,omitempty"`
	CurrentClaimObservedVRAM int64         `json:"current_claim_observed_vram_bytes,omitempty"`
	CurrentClaimObservedAt   *time.Time    `json:"current_claim_observed_at,omitempty"`
	CurrentClaimRung         bool          `json:"current_claim_rung"`
}

// ReadPlacementTable loads only receipts associated with the supplied claims,
// verifies their artifact links through the repository, and builds the current
// report. Artifact or storage corruption is returned as an error, never hidden
// behind an empty/unqualified table.
func ReadPlacementTable(ctx context.Context, repo PlacementQualificationRepository, claims []CapacityClaim, now time.Time, maxAge time.Duration) ([]PlacementTableRow, error) {
	ids := make([]string, 0, len(claims))
	for _, claim := range claims {
		if claim.ResourceKind == ResourceKindVRAM && claim.ClaimID != "" {
			ids = append(ids, claim.ClaimID)
		}
	}
	if len(ids) == 0 {
		return BuildPlacementTable(claims, nil, now, maxAge), nil
	}
	receipts, err := repo.ListPlacementQualifications(ctx, PlacementQualificationFilter{ClaimIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("load placement table receipts: %w", err)
	}
	return BuildPlacementTable(claims, receipts, now, maxAge), nil
}

// BuildPlacementTable combines immutable qualification receipts with the
// current claims. It emits an explicit missing row for an unqualified active
// role; it does not infer CPU versus unloaded from equal zero-byte rungs.
func BuildPlacementTable(claims []CapacityClaim, receipts []PlacementQualification, now time.Time, maxAge time.Duration) []PlacementTableRow {
	if maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour
	}
	claimByID := make(map[string]CapacityClaim, len(claims))
	for _, claim := range claims {
		claimByID[claim.ClaimID] = claim
	}
	seenCurrent := map[string]bool{}
	rows := make([]PlacementTableRow, 0, len(receipts)+len(claims))
	for _, q := range receipts {
		row := placementTableRowFromReceipt(q)
		claim, ok := claimByID[q.ClaimID]
		if !ok || claim.Generation != q.ClaimGeneration || claim.DegradeProfile == nil || claim.DegradeProfile.Role != q.Role || claim.DegradeProfile.Model != q.Model {
			row.EvidenceStatus = PlacementEvidenceMismatched
			row.Reason = "receipt does not match the current claim generation, role, and model"
		} else if now.Before(q.SampledAt) || now.Sub(q.SampledAt) > maxAge {
			row.EvidenceStatus = PlacementEvidenceStale
			row.Reason = "suite measurement is outside the freshness window"
		} else if !q.Passed || q.P95LatencyMS > q.MaxLatencyMS {
			row.EvidenceStatus = PlacementEvidenceFailed
			row.Reason = "suite failed or p95 exceeded the role latency objective"
		} else if step, found := findStep(claim.DegradeProfile, q.Rung); !found || !qualificationValid(q, step, now, maxAge) {
			row.EvidenceStatus = PlacementEvidenceFailed
			row.Reason = "measurement does not qualify the declared rung safety envelope"
		} else {
			row.EvidenceStatus = PlacementEvidenceQualified
			if claim.AmountBytes == step.AmountBytes && step.AmountBytes > 0 {
				row.CurrentClaimRung = true
			}
		}
		if ok {
			row.CurrentClaimGeneration = claim.Generation
			row.CurrentClaimAmountBytes = claim.AmountBytes
			row.CurrentClaimObservedVRAM = claim.ObservedBytes
			row.CurrentClaimObservedAt = claim.ObservedAt
		}
		if row.EvidenceStatus == PlacementEvidenceQualified && row.CurrentClaimRung {
			seenCurrent[q.ClaimID] = true
		}
		rows = append(rows, row)
	}
	for _, claim := range claims {
		if claim.ResourceKind != ResourceKindVRAM || !IsActiveClaimStatus(claim.Status) || seenCurrent[claim.ClaimID] {
			continue
		}
		role := "unmapped:" + claim.OwnerID
		model := ModelIdentity{}
		if claim.DegradeProfile != nil {
			if claim.DegradeProfile.Role != "" {
				role = claim.DegradeProfile.Role
			}
			model = claim.DegradeProfile.Model
		}
		rows = append(rows, PlacementTableRow{
			ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Role: role,
			Model: model, Device: "unknown", Rung: "unknown",
			EvidenceStatus: PlacementEvidenceMissing, Reason: "no fresh passing receipt identifies this claim's current role, device, and rung",
			CurrentClaimGeneration: claim.Generation, CurrentClaimAmountBytes: claim.AmountBytes,
			CurrentClaimObservedVRAM: claim.ObservedBytes, CurrentClaimObservedAt: claim.ObservedAt,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Role != rows[j].Role {
			return rows[i].Role < rows[j].Role
		}
		if rows[i].Rung != rows[j].Rung {
			return rows[i].Rung < rows[j].Rung
		}
		if rows[i].SampledAt.Equal(rows[j].SampledAt) {
			return rows[i].ReceiptID < rows[j].ReceiptID
		}
		return rows[i].SampledAt.After(rows[j].SampledAt)
	})
	return rows
}

func placementTableRowFromReceipt(q PlacementQualification) PlacementTableRow {
	return PlacementTableRow{
		ReceiptID: q.ReceiptID, ClaimID: q.ClaimID, ClaimGeneration: q.ClaimGeneration,
		Role: q.Role, Model: q.Model, Rung: q.Rung, Device: q.Device,
		Suite: q.Suite, SuiteDigest: q.SuiteDigest, RunID: q.RunID,
		ArtifactURI: q.ArtifactURI, ArtifactSHA: q.ArtifactSHA, Cases: q.Cases,
		Passed: q.Passed, P95LatencyMS: q.P95LatencyMS, MaxLatencyMS: q.MaxLatencyMS,
		MeasuredVRAMBytes: q.ObservedVRAM, SampledAt: q.SampledAt,
	}
}
