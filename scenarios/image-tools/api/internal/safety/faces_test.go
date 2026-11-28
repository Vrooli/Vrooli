package safety

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeFaces struct {
	n     int
	err   error
	calls int
}

func (f *fakeFaces) CountFaces(context.Context, [][]byte) (int, error) {
	f.calls++
	return f.n, f.err
}

func countOf(n int, err error) FaceCountFunc {
	return func(context.Context) (int, error) { return n, err }
}

// The consent rule protects the people in an image, so a low-weight request to a
// provider is asked for consent only when the image shows people. The rest of the
// table is what the job records as consent_basis.
func TestLowWeightConsentFollowsThePeopleInTheImage(t *testing.T) {
	gate := NewGate(TierLocal, nil)
	cases := []struct {
		name     string
		tier     string
		faces    FaceCountFunc
		required bool
		detail   string
	}{
		{"cloud, nobody shown", "byok-cloud", countOf(0, nil), false, "faces_detected=0"},
		{"cloud, one person", "byok-cloud", countOf(1, nil), true, "faces_detected=1"},
		{"cloud, a group", "byok-cloud", countOf(5, nil), true, "faces_detected=5"},
		{"cloud, detector fails", "byok-cloud", countOf(0, errors.New("boom")), true, ConsentDetailDetectorUnavailable},
		{"cloud, nobody can look", "byok-cloud", nil, true, ConsentDetailDetectorUnavailable},
		{"local never looks", "local-gpu", countOf(3, nil), false, ConsentDetailLocal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := gate.ConsentForInput(context.Background(), WeightLow, "caption", tc.tier, tc.faces)
			if got.Required != tc.required || got.Detail != tc.detail {
				t.Fatalf("required %v detail %q, want %v %q (%+v)", got.Required, got.Detail, tc.required, tc.detail, got)
			}
			if got.Required && (got.Basis != ConsentBasisRoute || got.Reason == "") {
				t.Errorf("a required consent names the route rule and explains itself: %+v", got)
			}
		})
	}
}

func TestLocalRouteDoesNotRunThePeopleCheck(t *testing.T) {
	counter := &fakeFaces{n: 2}
	gate := NewGate(TierLocal, nil)
	gate.SetFaceCounter(counter)
	called := false
	look := func(ctx context.Context) (int, error) { called = true; return gate.CountFaces(ctx, nil) }
	gate.ConsentForInput(context.Background(), WeightLow, "caption", "local-cpu", look)
	if called || counter.calls != 0 {
		t.Fatal("a route that stays on this machine must not pay for a face check")
	}
}

// Faces never loosen identity-altering work, whatever the check finds.
func TestHighWeightConsentIgnoresThePeopleCheck(t *testing.T) {
	gate := NewGate(TierLocal, nil)
	got := gate.ConsentForInput(context.Background(), WeightHigh, "face_swap", "byok-cloud", countOf(0, nil))
	if !got.Required || got.Basis != ConsentBasisRoute {
		t.Fatalf("a high-weight cloud route asks for consent with no face found: %+v", got)
	}
	if got.Detail != "" {
		t.Errorf("a rule that never looked has no people detail: %q", got.Detail)
	}
}

func TestEvaluateRouteInputRecordsWhyConsentWasOrWasNotNeeded(t *testing.T) {
	gate := NewGate(TierLocal, nil)
	ctx := context.Background()

	d := gate.EvaluateRouteInput(ctx, WeightLow, "caption", "byok-cloud", false, countOf(0, nil))
	if !d.Allowed || d.RecordConsent || d.ConsentBasis != "faces_detected=0" {
		t.Errorf("no people: allowed with no affirmation, basis recorded: %+v", d)
	}
	d = gate.EvaluateRouteInput(ctx, WeightLow, "caption", "byok-cloud", false, countOf(1, nil))
	if d.Allowed || !strings.Contains(d.Reason, "people") || d.ConsentBasis != "faces_detected=1" {
		t.Errorf("a person without consent is refused: %+v", d)
	}
	d = gate.EvaluateRouteInput(ctx, WeightLow, "caption", "byok-cloud", true, countOf(1, nil))
	if !d.Allowed || !d.RecordConsent {
		t.Errorf("an affirmed person is admitted and the affirmation is logged: %+v", d)
	}
	d = gate.EvaluateRouteInput(ctx, WeightLow, "caption", "byok-cloud", false, countOf(0, errors.New("down")))
	if d.Allowed || d.ConsentBasis != ConsentDetailDetectorUnavailable {
		t.Errorf("an unavailable detector fails closed: %+v", d)
	}
	d = gate.EvaluateRouteInput(ctx, WeightLow, "caption", "local-gpu", false, nil)
	if !d.Allowed || d.ConsentBasis != ConsentDetailLocal {
		t.Errorf("a local run is admitted with its basis recorded: %+v", d)
	}
	d = gate.EvaluateRouteInput(ctx, WeightHigh, "face_swap", "byok-cloud", true, nil)
	if !d.Allowed || d.ConsentBasis != ConsentBasisRoute {
		t.Errorf("a high-weight route names the rule that asked: %+v", d)
	}
}

// Without a detector "nobody is shown" cannot be established: the request keeps the
// requirement, and so does a caller that never looked (the plain EvaluateRoute).
func TestNoFaceCounterFailsClosed(t *testing.T) {
	gate := NewGate(TierLocal, nil)
	if _, err := gate.CountFaces(context.Background(), [][]byte{{1}}); !errors.Is(err, ErrNoFaceCounter) {
		t.Fatalf("CountFaces without a counter = %v, want ErrNoFaceCounter", err)
	}
	if d := gate.EvaluateRoute(WeightLow, "caption", "byok-cloud", false); d.Allowed {
		t.Errorf("EvaluateRoute never looks at an image, so a cloud route still asks: %+v", d)
	}
}

func fakeCounterProgram(t *testing.T, body string) *CommandFaceCounter {
	t.Helper()
	path := filepath.Join(t.TempDir(), "count.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ncat >/dev/null\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &CommandFaceCounter{program: path, timeout: 5 * time.Second, maxBytes: 1 << 20}
}

func TestCommandFaceCounterSumsTheCountsOfEveryImage(t *testing.T) {
	c := fakeCounterProgram(t, `echo '{"counts":[1,0,2]}'`)
	n, err := c.CountFaces(context.Background(), [][]byte{{1}, {2}, {3}})
	if err != nil || n != 3 {
		t.Fatalf("CountFaces = %d, %v; want 3", n, err)
	}
}

// Every one of these must surface as an error: a count of zero clears the consent
// rule, so a broken program may never read as "nobody is shown".
func TestCommandFaceCounterErrorsAreNeverACountOfZero(t *testing.T) {
	cases := map[string]string{
		"non-zero exit":      `echo broken >&2; exit 1`,
		"malformed output":   `echo not-json`,
		"too few answers":    `echo '{"counts":[0]}'`,
		"negative count":     `echo '{"counts":[-1,0]}'`,
		"empty output":       `true`,
		"missing key":        `echo '{}'`,
		"wrong answer shape": `echo '{"counts":"0"}'`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			n, err := fakeCounterProgram(t, body).CountFaces(context.Background(), [][]byte{{1}, {2}})
			if err == nil {
				t.Fatalf("CountFaces = %d with no error", n)
			}
		})
	}
}

func TestCommandFaceCounterNamesMissingWeightsAndTimeouts(t *testing.T) {
	_, err := fakeCounterProgram(t, `echo "face_count: missing-weights=yunet" >&2; exit 4`).CountFaces(context.Background(), [][]byte{{1}})
	var missing *MissingWeightsError
	if !errors.As(err, &missing) || len(missing.IDs) != 1 || missing.IDs[0] != "yunet" {
		t.Fatalf("err = %v, want MissingWeightsError for yunet", err)
	}
	slow := fakeCounterProgram(t, `sleep 5`)
	slow.timeout = 100 * time.Millisecond
	if _, err := slow.CountFaces(context.Background(), [][]byte{{1}}); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want a timeout", err)
	}
}

func TestBundledFaceCounterNeedsItsLocations(t *testing.T) {
	if _, err := NewBundledFaceCounter(BundledReviewerConfig{}); err == nil {
		t.Fatal("a counter without python, sidecar or models root must not build")
	}
}
