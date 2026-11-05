package evidence

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/certification"

	"github.com/vrooli/api-core/receiptsigning"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var (
	testNow     = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	releaseA    = "sha256:" + strings.Repeat("a", 64)
	releaseB    = "sha256:" + strings.Repeat("b", 64)
	targetA     = "machine:fixture-a"
	targetOther = "host:203.0.113.99"
)

func testIdentity() ReviewIdentity {
	return ReviewIdentity{
		ScenarioID: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ReleaseDigest: releaseA,
		ConfigurationDigest: "sha256:" + strings.Repeat("c", 64), TargetSet: []string{targetA}, Environment: "staging",
		Channel: "stable", PolicyVersion: 2,
	}
}

func mustProfile(t *testing.T) *Profile {
	t.Helper()
	p, err := LoadProfile(ProfileCloudLaunchV1)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func record(id string, cell CellID, d Disposition, release, target string, at time.Time) Record {
	r := Record{
		SchemaVersion: RecordSchemaVersion, ID: id, ProfileID: ProfileCloudLaunchV1, CaseID: cell.CaseID, Lane: cell.Lane, Disposition: d,
		Binding:    Binding{ProducerRef: ProducerRef, DeploymentID: "dep-1", ReleaseDigest: release, TargetKey: target, OperationID: "op-" + id},
		ObservedAt: at, RecordedAt: at,
	}
	if d == DispositionPassed {
		r.ReceiptRefs = []string{"cloud-target:receipt:" + id}
	} else {
		r.Reason = string(d) + " by fixture"
	}
	return r
}

func passAll(p *Profile, release, target string, at time.Time) []Record {
	out := make([]Record, 0, len(p.Cells))
	for i, cell := range p.Cells {
		out = append(out, record("r"+strings.Repeat("0", 2)+cell.String()+string(rune('a'+i%26)), cell, DispositionPassed, release, target, at))
	}
	return out
}

// TestProfileIsTheReleaseLaneOfTheMatrixWithoutPackagingAssumptions
// [REQ:STC-P0-035] proves P17-O01: the profile is drawn from the matrix and
// keeps every declared lane per case.
func TestProfileIsTheReleaseLaneOfTheMatrixWithoutPackagingAssumptions(t *testing.T) {
	p := mustProfile(t)
	if len(CloudLaunchCases) != 17 {
		t.Fatalf("profile cases = %d", len(CloudLaunchCases))
	}
	if p.MatrixRevision != "cloud-launch-v1" {
		t.Fatalf("matrix revision = %q", p.MatrixRevision)
	}
	for _, id := range CloudLaunchCases {
		c, ok := p.Case(id)
		if !ok || !c.Required {
			t.Fatalf("case %s missing or optional", id)
		}
		for _, lane := range c.Lanes {
			if !p.Contains(CellID{CaseID: id, Lane: lane}) {
				t.Fatalf("cell %s/%s missing", id, lane)
			}
		}
	}
	for _, cell := range p.Cells {
		c, _ := p.Case(cell.CaseID)
		for _, dim := range c.Dimensions {
			if strings.Contains(strings.ToLower(dim), "electron") || strings.Contains(strings.ToLower(dim), "installer") {
				t.Fatalf("cell %s carries a packaging dimension %q", cell, dim)
			}
		}
	}
	if len(p.Lanes[certification.LaneAPI]) != 8 {
		t.Fatalf("api lane cells = %d, expected the eight GOV cells", len(p.Lanes[certification.LaneAPI]))
	}
	if _, err := LoadProfile("desktop-launch-v1"); err == nil {
		t.Fatal("unknown profile must be refused")
	}
}

// TestEvaluateKeepsTheSharedVocabularyDistinct [REQ:STC-P0-035] proves
// GOV-01 (missing cell blocks), GOV-02 (evidence for another release or
// target is incompatible), GOV-07 (unavailable stays unavailable) and
// P17-A04 (a rerun never masks the failed record until withdrawn).
func TestEvaluateKeepsTheSharedVocabularyDistinct(t *testing.T) {
	p := mustProfile(t)
	full := passAll(p, releaseA, targetA, testNow)
	s := Evaluate(p, full, releaseA, targetA, testNow)
	if !s.Passed || len(s.BlockingCells) != 0 || s.RequiredCells != len(p.Cells) {
		t.Fatalf("complete evidence must pass: %+v", s.BlockingCells)
	}

	// GOV-01: drop one required cell.
	missingOne := full[1:]
	s = Evaluate(p, missingOne, releaseA, targetA, testNow)
	if s.Passed || len(s.BlockingCells) != 1 || s.BlockingCells[0] != p.Cells[0].String() {
		t.Fatalf("missing cell must block: %+v", s.BlockingCells)
	}
	if s.Cells[0].Disposition != DispositionMissing {
		t.Fatalf("cell disposition = %s", s.Cells[0].Disposition)
	}

	// GOV-02: the dropped cell exists only for another release / another target.
	foreignRelease := append(append([]Record(nil), missingOne...), record("fr", p.Cells[0], DispositionPassed, releaseB, targetA, testNow))
	s = Evaluate(p, foreignRelease, releaseA, targetA, testNow)
	if s.Passed || s.Cells[0].Disposition != DispositionMissing || !strings.Contains(s.Cells[0].Reason, "incompatible") {
		t.Fatalf("foreign release evidence must be incompatible: %+v", s.Cells[0])
	}
	foreignTarget := append(append([]Record(nil), missingOne...), record("ft", p.Cells[0], DispositionPassed, releaseA, targetOther, testNow))
	s = Evaluate(p, foreignTarget, releaseA, targetA, testNow)
	if s.Passed || s.Cells[0].Disposition != DispositionMissing || !strings.Contains(s.Cells[0].Reason, targetOther) {
		t.Fatalf("foreign target evidence must be incompatible: %+v", s.Cells[0])
	}

	// GOV-07: an unavailable record is reported as unavailable, never pass.
	unavailable := append(append([]Record(nil), missingOne...), record("un", p.Cells[0], DispositionUnavailable, releaseA, targetA, testNow))
	s = Evaluate(p, unavailable, releaseA, targetA, testNow)
	if s.Passed || s.Cells[0].Disposition != DispositionUnavailable {
		t.Fatalf("unavailable must stay unavailable: %+v", s.Cells[0])
	}

	// P17-A04: failed then rerun passed keeps failed until withdrawn.
	failed := record("f1", p.Cells[0], DispositionFailed, releaseA, targetA, testNow)
	rerun := record("f2", p.Cells[0], DispositionPassed, releaseA, targetA, testNow.Add(time.Minute))
	s = Evaluate(p, append(append([]Record(nil), missingOne...), failed, rerun), releaseA, targetA, testNow)
	if s.Passed || s.Cells[0].Disposition != DispositionFailed || s.Cells[0].RecordID != "f1" {
		t.Fatalf("rerun must not mask the failed record: %+v", s.Cells[0])
	}
	withdrawal := record("w1", p.Cells[0], DispositionSkipped, releaseA, targetA, testNow.Add(2*time.Minute))
	withdrawal.Withdraws = "f1"
	withdrawal.Reason = "owner withdrew f1: fixture host lost power during the run"
	s = Evaluate(p, append(append([]Record(nil), missingOne...), failed, rerun, withdrawal), releaseA, targetA, testNow)
	if !s.Passed || s.Cells[0].RecordID != "f2" {
		t.Fatalf("withdrawn failure lets the rerun count: %+v", s.Cells[0])
	}

	// The empty candidate can never pass.
	if Evaluate(p, full, "", targetA, testNow).Passed {
		t.Fatal("empty release digest must not pass")
	}
}

// TestRecordValidationAndSignature [REQ:STC-P0-035] proves P17 step 12: a
// record without producer binding or receipt refs is refused and a tampered
// signed record fails verification.
func TestRecordValidationAndSignature(t *testing.T) {
	p := mustProfile(t)
	r := record("r1", p.Cells[0], DispositionPassed, releaseA, targetA, testNow)
	if err := r.Validate(p); err != nil {
		t.Fatal(err)
	}
	unbound := r
	unbound.Binding.OperationID = ""
	if err := unbound.Validate(p); err == nil {
		t.Fatal("record without operation or observation must be refused")
	}
	noRefs := r
	noRefs.ReceiptRefs = nil
	if err := noRefs.Validate(p); err == nil {
		t.Fatal("passed record without receipt refs must be refused")
	}
	foreignProducer := r
	foreignProducer.Binding.ProducerRef = "caller"
	if err := foreignProducer.Binding.Check(Expect{DeploymentID: "dep-1"}); err == nil {
		t.Fatal("foreign producer must be refused")
	}
	if err := r.Binding.Check(Expect{ReleaseDigest: releaseB}); err == nil {
		t.Fatal("wrong release binding must be refused")
	}
	signer := receiptsigning.NewDevelopmentSigner()
	if err := r.Sign(context.Background(), signer); err != nil {
		t.Fatal(err)
	}
	if err := r.VerifySignature(context.Background(), signer); err != nil {
		t.Fatal(err)
	}
	tampered := r
	tampered.Disposition = DispositionFailed
	if err := tampered.VerifySignature(context.Background(), signer); err == nil {
		t.Fatal("tampered record must fail verification")
	}
	unsigned := record("r2", p.Cells[0], DispositionPassed, releaseA, targetA, testNow)
	if err := unsigned.VerifySignature(context.Background(), signer); err != ErrUnsigned {
		t.Fatalf("unsigned record error = %v", err)
	}
}

// TestIdentityDigestBindsEveryFacet [REQ:STC-P0-035] proves P17 step 7.
func TestIdentityDigestBindsEveryFacet(t *testing.T) {
	base := testIdentity()
	digest, err := base.Digest()
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*ReviewIdentity){
		"commit":        func(i *ReviewIdentity) { i.CandidateCommit = "def456" },
		"release":       func(i *ReviewIdentity) { i.ReleaseDigest = releaseB },
		"configuration": func(i *ReviewIdentity) { i.ConfigurationDigest = "sha256:" + strings.Repeat("d", 64) },
		"targets":       func(i *ReviewIdentity) { i.TargetSet = []string{targetA, targetOther} },
		"environment":   func(i *ReviewIdentity) { i.Environment = "production" },
		"channel":       func(i *ReviewIdentity) { i.Channel = "beta" },
		"policy":        func(i *ReviewIdentity) { i.PolicyVersion = 3 },
	}
	for name, mutate := range mutations {
		changed := testIdentity()
		mutate(&changed)
		other, err := changed.Digest()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if other == digest {
			t.Fatalf("%s change must alter the identity digest", name)
		}
	}
	reordered := testIdentity()
	reordered.TargetSet = []string{" " + targetA + " ", targetA}
	if same, _ := reordered.Digest(); same != digest {
		t.Fatal("target order and whitespace must not change the digest")
	}
	partial := testIdentity()
	partial.CandidateID = "candidate-1"
	if _, err := partial.Digest(); err == nil {
		t.Fatal("partial release binding must be refused")
	}
}

// TestDecideRefusesEverythingButTheExactApprovedReview [REQ:STC-P0-035]
// proves GOV-03 (identity mismatch) and GOV-06 (revoked or superseded
// approval before the mutation).
func TestDecideRefusesEverythingButTheExactApprovedReview(t *testing.T) {
	identity := testIdentity()
	digest, _ := identity.Digest()
	pub := &Publication{ID: "pub-1", Identity: identity, IdentityDigest: digest, ReviewRef: "rr-1", ReleaseDigest: releaseA}
	approved := &ReviewSnapshot{Key: "rr-1", Status: ReviewApproved, Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ArtifactDigest: releaseA, Targets: []string{targetA}, Channel: "stable", PolicyVersion: 2}
	if refusal := Decide(pub, approved, nil); refusal != nil {
		t.Fatalf("exact approved review must be accepted: %+v", refusal)
	}
	cases := map[string]struct {
		review *ReviewSnapshot
		err    error
		code   string
	}{
		"unavailable":  {nil, context.DeadlineExceeded, RefusalReviewUnavailable},
		"not approved": {&ReviewSnapshot{Key: "rr-1", Status: "agent_review", Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ArtifactDigest: releaseA, Targets: []string{targetA}, Channel: "stable", PolicyVersion: 2}, nil, RefusalReviewNotApproved},
		"superseded":   {&ReviewSnapshot{Key: "rr-1", Status: ReviewSuperseded, Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ArtifactDigest: releaseA, Targets: []string{targetA}, Channel: "stable", PolicyVersion: 2}, nil, RefusalReviewRevoked},
		"other review": {&ReviewSnapshot{Key: "rr-2", Status: ReviewApproved}, nil, RefusalReviewMismatch},
		"other bytes":  {&ReviewSnapshot{Key: "rr-1", Status: ReviewApproved, Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ArtifactDigest: releaseB, Targets: []string{targetA}, Channel: "stable", PolicyVersion: 2}, nil, RefusalReviewMismatch},
		"other target": {&ReviewSnapshot{Key: "rr-1", Status: ReviewApproved, Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ArtifactDigest: releaseA, Targets: []string{targetOther}, Channel: "stable", PolicyVersion: 2}, nil, RefusalReviewMismatch},
	}
	for name, tc := range cases {
		refusal := Decide(pub, tc.review, tc.err)
		if refusal == nil || refusal.Code != tc.code {
			t.Fatalf("%s: refusal = %+v, want %s", name, refusal, tc.code)
		}
	}
	// Approved bytes replaced after the request: the stored identity digest
	// no longer matches the identity the caller is trying to publish.
	replaced := *pub
	replaced.Identity.ReleaseDigest = releaseB
	if refusal := Decide(&replaced, approved, nil); refusal == nil || refusal.Code != RefusalReviewMismatch {
		t.Fatalf("replaced bytes must be refused: %+v", refusal)
	}
}

// TestRecordActivationUsesTheTargetReceipt [REQ:STC-P0-035] proves GOV-05
// and P17-A03: the published release is what the target reports, and the
// predecessor is resolved from the target pointer or published history.
func TestRecordActivationUsesTheTargetReceipt(t *testing.T) {
	identity := testIdentity()
	digest, _ := identity.Digest()
	earlier := testNow.Add(-time.Hour)
	history := []Publication{{ID: "pub-0", State: StatePublished, ActivatedReleaseDigest: releaseB, PublishedAt: &earlier}}
	pub := &Publication{ID: "pub-1", Identity: identity, IdentityDigest: digest, ReviewRef: "rr-1", ReleaseDigest: releaseA, State: StateActivating}
	RecordActivation(pub, TargetReceipt{ActiveRelease: strings.TrimPrefix(releaseA, "sha256:"), OperationID: "op-9", Fence: 3}, history, testNow)
	if pub.State != StatePublished || pub.ActivatedReleaseDigest != strings.TrimPrefix(releaseA, "sha256:") || pub.PredecessorReleaseDigest != releaseB || pub.TargetReceipt.ReceiptDigest == "" {
		t.Fatalf("publication = %+v", pub)
	}
	pointer := &Publication{ID: "pub-2", Identity: identity, IdentityDigest: digest, ReleaseDigest: releaseA, State: StateActivating}
	RecordActivation(pointer, TargetReceipt{ActiveRelease: releaseA, PreviousRelease: "prev-from-target"}, history, testNow)
	if pointer.PredecessorReleaseDigest != "prev-from-target" {
		t.Fatalf("target pointer must win: %q", pointer.PredecessorReleaseDigest)
	}
	mismatch := &Publication{ID: "pub-3", Identity: identity, IdentityDigest: digest, ReleaseDigest: releaseA, State: StateActivating}
	RecordActivation(mismatch, TargetReceipt{ActiveRelease: releaseB}, nil, testNow)
	if mismatch.State != StateFailed || mismatch.Refusal == nil || mismatch.Refusal.Code != RefusalTargetMismatch || mismatch.PublishedAt != nil {
		t.Fatalf("mismatched target release must fail: %+v", mismatch)
	}
}

// TestCoverageVerdictReferencesEveryCellAndNeverPassesEarly [REQ:STC-P0-035]
// proves the DM-facing verdict carries refs per cell and the target key.
func TestCoverageVerdictReferencesEveryCellAndNeverPassesEarly(t *testing.T) {
	p := mustProfile(t)
	s := Evaluate(p, passAll(p, releaseA, targetA, testNow), releaseA, targetA, testNow)
	verdict, err := s.CoverageVerdict("pub-1", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Disposition != commonv1.Disposition_DISPOSITION_PASSED || verdict.Target.Platform != targetA || len(verdict.Refs) != len(p.Cells) {
		t.Fatalf("verdict = %+v", verdict)
	}
	partial := Evaluate(p, passAll(p, releaseA, targetA, testNow)[1:], releaseA, targetA, testNow)
	verdict, _ = partial.CoverageVerdict("pub-1", testNow)
	if verdict.Disposition != commonv1.Disposition_DISPOSITION_FAILED || len(verdict.Refs) != len(p.Cells)-1 {
		t.Fatalf("partial verdict = %+v", verdict)
	}
	if _, err := s.CoverageVerdict("", testNow); err == nil {
		t.Fatal("verdict without run id must be refused")
	}
}
