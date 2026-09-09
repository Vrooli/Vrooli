package learning

import (
	"testing"
	"time"

	source "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var start = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func attempt(id, task string, n int32, outcome string) *pb.Attempt {
	a := &pb.Attempt{AttemptId: id, TaskId: task, AttemptNumber: n, TaskStartedAt: start.Format(time.RFC3339), Operation: "inspect", ContextKey: "linux:tool-v1", StartedAt: start.Add(time.Duration(n-1) * time.Minute).Format(time.RFC3339), FinishedAt: start.Add(time.Duration(n) * time.Minute).Format(time.RFC3339), Outcome: outcome, RecallStatus: "no_match", Provenance: "operator", Trigger: "inspect pipeline", Approach: "pipeline-inspect"}
	if outcome == "failed" {
		a.FailureFingerprint = "timeout"
	}
	if outcome == "verified_success" {
		a.EvidenceRefs = []string{"run:1"}
	}
	return a
}

func entry(t *testing.T, a *pb.Attempt) *source.Entry {
	t.Helper()
	body, err := Encode(a)
	if err != nil {
		t.Fatal(err)
	}
	return &source.Entry{Id: a.AttemptId, Body: body, Kind: "task-record", CreatedAt: timestamppb.New(start.Add(time.Hour))}
}

func measure(es []*source.Entry) *pb.MeasureLearningResponse {
	return Measure(es, "scope", start, start.Add(24*time.Hour), "", "", false)
}

func TestOutcomeMetricsKeepUnresolvedAndUnknownVisible(t *testing.T) {
	a := attempt("a", "task", 1, "failed")
	a.RecallStatus = "matched"
	a.Advice = []*pb.AdviceUse{{EntryId: "prior", Decision: "applied", DecisionChange: "used recommended route", Verdict: "contradicted", EvidenceRefs: []string{"run:failed"}}}
	es := []*source.Entry{entry(t, a), entry(t, attempt("b", "task", 2, "verified_success")), entry(t, attempt("c", "other", 1, "failed")), entry(t, attempt("d", "down", 1, "unavailable"))}
	r := measure(es)
	if !r.Reliable || len(r.Cohorts) != 1 {
		t.Fatalf("%+v", r)
	}
	c := r.Cohorts[0]
	if c.Attempts != 4 || c.CompletedTasks != 1 || c.UnresolvedTasks != 2 || c.RecurringFailureFingerprints != 1 || c.RepeatedFailures != 1 || c.Unavailable != 1 || c.GetMedianAttemptsToSuccess() != 2 || c.GetMedianSecondsToSuccess() != 120 || c.ContradictionRate == nil || *c.ContradictionRate != 1 {
		t.Fatalf("%+v", c)
	}
}

func TestEmptyAndUnassessedAreNotZeroSuccess(t *testing.T) {
	r := measure(nil)
	if r.Reliable || r.Reason != "unreliable:no_eligible_attempts" {
		t.Fatal(r)
	}
	r = measure([]*source.Entry{entry(t, attempt("a", "task", 1, "unknown"))})
	c := r.Cohorts[0]
	if c.MedianSecondsToSuccess != nil || c.ContradictionRate != nil || c.UnresolvedTasks != 1 {
		t.Fatal(c)
	}
}

func TestExposedAdviceIsNeitherAppliedNorRejected(t *testing.T) {
	a := attempt("exposure", "task", 1, "unknown")
	a.RecallStatus = "matched"
	a.Advice = []*pb.AdviceUse{{EntryId: "prior", Decision: "unassessed", Verdict: "unknown"}}
	c := measure([]*source.Entry{entry(t, a)}).Cohorts[0]
	if c.AppliedAdvice != 0 || c.RejectedAdvice != 0 || c.UnassessedAdvice != 1 {
		t.Fatalf("exposure fabricated a decision: %+v", c)
	}
	for _, mutate := range []func(*pb.AdviceUse){
		func(u *pb.AdviceUse) { u.DecisionChange = "changed route" },
		func(u *pb.AdviceUse) { u.Verdict = "supported" },
		func(u *pb.AdviceUse) { u.EvidenceRefs = []string{"run:1"} },
	} {
		b := proto.Clone(a).(*pb.Attempt)
		mutate(b.Advice[0])
		if Validate(b) == nil {
			t.Fatal("unassessed exposure must not assert a decision or support")
		}
	}
}

func TestContextSeparationTestExclusionAndReplayDeduplication(t *testing.T) {
	a := attempt("a", "task", 1, "verified_success")
	b := proto.Clone(a).(*pb.Attempt)
	b.AttemptId = "b"
	b.ContextKey = "windows:tool-v1"
	x := proto.Clone(a).(*pb.Attempt)
	x.AttemptId = "test"
	x.Provenance = "test"
	r := measure([]*source.Entry{entry(t, a), entry(t, b), entry(t, x), entry(t, a)})
	if len(r.Cohorts) != 2 || r.EligibleAttempts != 2 || r.ExcludedTestAttempts != 1 || r.DuplicateAttempts != 1 {
		t.Fatal(r)
	}
}

func TestIncompleteHistoryCannotProduceEffortMedian(t *testing.T) {
	a := attempt("a", "task", 2, "verified_success")
	r := measure([]*source.Entry{entry(t, a)})
	if r.Cohorts[0].MedianSecondsToSuccess != nil || r.Cohorts[0].LeftCensoredTasks != 1 {
		t.Fatal(r)
	}
	a = attempt("b", "task", 1, "verified_success")
	a.TaskStartedAt = start.Add(-time.Hour).Format(time.RFC3339)
	r = measure([]*source.Entry{entry(t, a)})
	if r.Cohorts[0].MedianSecondsToSuccess != nil {
		t.Fatal(r)
	}
}

func TestHalfOpenWindowAndScanLimit(t *testing.T) {
	a := attempt("a", "task", 1, "verified_success")
	es := []*source.Entry{entry(t, a)}
	end, _ := time.Parse(time.RFC3339, a.FinishedAt)
	r := Measure(es, "scope", start, end, "", "", false)
	if r.EligibleAttempts != 0 {
		t.Fatal(r)
	}
	r = Measure(es, "scope", start, start.Add(24*time.Hour), "", "", true)
	if r.Reliable || r.Reason != "unreliable:scan_limit" {
		t.Fatal(r)
	}
}

func TestLegacyInvalidAndConflictingRecordsDoNotLookHealthy(t *testing.T) {
	a := attempt("a", "task", 1, "verified_success")
	e := entry(t, a)
	legacy := &source.Entry{Id: "old", Kind: "task-record", Body: "old prose", CreatedAt: e.CreatedAt}
	r := measure([]*source.Entry{e, legacy})
	if r.Reliable || r.LegacyTaskRecords != 1 {
		t.Fatal(r)
	}
	bad := *e
	bad.Body = Prefix + "broken"
	r = measure([]*source.Entry{&bad})
	if r.InvalidRecords != 1 || r.Reliable {
		t.Fatal(r)
	}
	b := proto.Clone(a).(*pb.Attempt)
	b.Approach = "different"
	r = measure([]*source.Entry{e, entry(t, b)})
	if r.InvalidRecords != 1 || r.DuplicateAttempts != 1 || r.Reliable {
		t.Fatal(r)
	}
}

func TestValidationRejectsUnattributableClaims(t *testing.T) {
	cases := map[string]func(*pb.Attempt){"no evidence": func(a *pb.Attempt) { a.EvidenceRefs = nil }, "bad times": func(a *pb.Attempt) { a.FinishedAt = "yesterday" }, "no attempt ordinal": func(a *pb.Attempt) { a.AttemptNumber = 0 }, "no task identity": func(a *pb.Attempt) { a.TaskId = "" }, "missing fingerprint": func(a *pb.Attempt) { a.Outcome = "failed" }, "unavailable recall with advice": func(a *pb.Attempt) { a.RecallStatus = "unavailable"; a.Advice = []*pb.AdviceUse{{EntryId: "prior"}} }, "unsupported verdict": func(a *pb.Attempt) {
		a.RecallStatus = "matched"
		a.Advice = []*pb.AdviceUse{{EntryId: "prior", Decision: "applied", DecisionChange: "route", Verdict: "supported"}}
	}}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			a := attempt("a", "t", 1, "verified_success")
			mutate(a)
			if Validate(a) == nil {
				t.Fatal("accepted invalid attempt")
			}
		})
	}
}

func TestObservationRoundTripAndUnknownNeedsNoEvidence(t *testing.T) {
	o := &pb.Observation{ObservationId: "observation-1", AttemptId: "attempt-1", Disposition: "supported", EvidenceRefs: []string{"receipt:r1"}, MethodRevision: "method:v2", Provenance: "operator", ObservedAt: "2026-09-01T00:00:00Z"}
	body, err := EncodeObservation(o)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeObservation(body)
	if err != nil || got.ObservationId != o.ObservationId || got.MethodRevision != o.MethodRevision {
		t.Fatalf("round trip changed observation: %#v (%v)", got, err)
	}
	o.Disposition = "unknown"
	o.EvidenceRefs = nil
	if _, err := EncodeObservation(o); err != nil {
		t.Fatalf("unknown observation should remain recordable without evidence: %v", err)
	}
}

func TestObservationRejectsUnsupportedDisposition(t *testing.T) {
	o := &pb.Observation{ObservationId: "observation-1", AttemptId: "attempt-1", Disposition: "maybe", Provenance: "operator", ObservedAt: "2026-09-01T00:00:00Z"}
	if _, err := EncodeObservation(o); err == nil {
		t.Fatal("accepted unsupported disposition")
	}
}

func TestMeasureProjectsLatestFeedbackWithoutChangingAttempt(t *testing.T) {
	a := attempt("attempt-1", "task-1", 1, "unknown")
	o := &pb.Observation{ObservationId: "observation-1", AttemptId: a.AttemptId, Disposition: "supported", EvidenceRefs: []string{"receipt:r1"}, Provenance: "operator", ObservedAt: "2026-09-02T00:00:00Z"}
	body, err := EncodeObservation(o)
	if err != nil {
		t.Fatal(err)
	}
	result := measure([]*source.Entry{entry(t, a), &source.Entry{Id: o.ObservationId, Kind: "attempt-observation", Body: body, CreatedAt: timestamppb.New(start.Add(2 * time.Hour))}})
	if len(result.Cohorts) != 1 || result.Cohorts[0].SupportedFeedback != 1 || result.Cohorts[0].Unknown != 1 {
		t.Fatalf("feedback was not projected alongside original attempt: %+v", result.Cohorts)
	}
}

func TestAgentEffortKeepsMissingDistinctFromZero(t *testing.T) {
	a := attempt("measured", "one", 1, "verified_success")
	a.FirstActionAt = proto.String(start.Add(5 * time.Second).Format(time.RFC3339))
	a.ToolRoundTrips = proto.Int32(2)
	a.VisualReasoningCalls = proto.Int32(0)
	a.ReusedWorkflow = proto.Bool(true)
	b := attempt("unknown", "two", 1, "unknown")
	c := measure([]*source.Entry{entry(t, a), entry(t, b)}).Cohorts[0]
	if c.FirstActionSamples != 1 || c.ToolRoundTripSamples != 1 || c.VisualReasoningSamples != 1 || c.ReuseSamples != 1 {
		t.Fatalf("missing values entered denominator: %+v", c)
	}
	if c.GetMedianSecondsToFirstAction() != 5 || c.GetMedianToolRoundTrips() != 2 || c.MedianVisualReasoningCalls == nil || c.GetMedianVisualReasoningCalls() != 0 || c.GetWorkflowReuseRate() != 1 {
		t.Fatalf("wrong metrics: %+v", c)
	}
	a.FirstActionAt = proto.String(start.Add(-time.Second).Format(time.RFC3339))
	if Validate(a) == nil {
		t.Fatal("accepted action before attempt")
	}
	a.FirstActionAt = nil
	a.ToolRoundTrips = proto.Int32(-1)
	if Validate(a) == nil {
		t.Fatal("accepted negative count")
	}
}

func TestFirstActionLatencyDoesNotCountRetryAsAnotherTask(t *testing.T) {
	first := attempt("first", "task", 1, "failed")
	first.FirstActionAt = proto.String(start.Add(5 * time.Second).Format(time.RFC3339))
	retry := attempt("retry", "task", 2, "verified_success")
	retry.FirstActionAt = proto.String(start.Add(90 * time.Second).Format(time.RFC3339))
	c := measure([]*source.Entry{entry(t, first), entry(t, retry)}).Cohorts[0]
	if c.FirstActionSamples != 1 || c.GetMedianSecondsToFirstAction() != 5 {
		t.Fatalf("retry distorted first action: %+v", c)
	}
}

func TestConflictingFirstAttemptsDoNotProduceLatency(t *testing.T) {
	first := attempt("one", "task", 1, "failed")
	first.FirstActionAt = proto.String(start.Add(5 * time.Second).Format(time.RFC3339))
	duplicate := attempt("two", "task", 1, "failed")
	duplicate.FirstActionAt = proto.String(start.Add(15 * time.Second).Format(time.RFC3339))
	c := measure([]*source.Entry{entry(t, first), entry(t, duplicate)}).Cohorts[0]
	if c.FirstActionSamples != 0 || c.MedianSecondsToFirstAction != nil {
		t.Fatalf("ambiguous task earned timing: %+v", c)
	}
}
