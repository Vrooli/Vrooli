package safety

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestMinorReviewerFromEnvUnsetIsNil(t *testing.T) {
	t.Setenv(MinorSafetyEnv, "")
	reviewer, err := MinorReviewerFromEnv()
	if err != nil || reviewer != nil {
		t.Fatalf("reviewer = %v, err = %v; want nil, nil", reviewer, err)
	}
}

func TestNewCommandReviewerRejectsEmpty(t *testing.T) {
	if _, err := NewCommandReviewer("   "); err == nil {
		t.Fatal("empty command accepted")
	}
}

// TestCommandReviewerContract re-executes the test binary as the reviewer so
// the documented stdin/stdout contract is exercised end to end without shelling
// out to an interpreter in the test.
func TestCommandReviewerContract(t *testing.T) {
	command := os.Args[0] + " -test.run=TestMinorReviewerHelperProcess"
	t.Setenv("GO_WANT_MINOR_REVIEWER", "1")
	t.Setenv("MINOR_REVIEWER_VERDICT", `{"minor":true,"sexual":true,"detail":"helper flagged"}`)

	reviewer, err := NewCommandReviewer(command)
	if err != nil {
		t.Fatal(err)
	}
	verdict, err := reviewer.Review(context.Background(), [][]byte{[]byte("\x89PNG\r\n\x1a\nrest")})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !verdict.Minor || !verdict.Sexual || verdict.Detail != "helper flagged" {
		t.Fatalf("verdict = %+v", verdict)
	}
}

func TestCommandReviewerRejectsMalformedOutput(t *testing.T) {
	t.Setenv("GO_WANT_MINOR_REVIEWER", "1")
	t.Setenv("MINOR_REVIEWER_VERDICT", "not json")
	reviewer, err := NewCommandReviewer(os.Args[0] + " -test.run=TestMinorReviewerHelperProcess")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reviewer.Review(context.Background(), [][]byte{[]byte("x")}); err == nil {
		t.Fatal("malformed reviewer output accepted")
	}
}

func TestCommandReviewerBoundsInput(t *testing.T) {
	reviewer, err := NewCommandReviewer(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	reviewer.maxBytes = 4
	if _, err := reviewer.Review(context.Background(), [][]byte{[]byte("too large")}); err == nil {
		t.Fatal("oversized review input accepted")
	}
}

// TestMinorReviewerHelperProcess is the reviewer subprocess. It verifies the
// request shape and echoes the verdict configured by the parent test.
func TestMinorReviewerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MINOR_REVIEWER") != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	var request minorReviewRequest
	if err := json.Unmarshal(raw, &request); err != nil || len(request.Images) != 1 {
		os.Exit(3)
	}
	if _, err := base64.StdEncoding.DecodeString(request.Images[0].Base64); err != nil {
		os.Exit(4)
	}
	if _, err := os.Stdout.WriteString(os.Getenv("MINOR_REVIEWER_VERDICT")); err != nil {
		os.Exit(5)
	}
	os.Exit(0)
}

func TestNewBundledReviewerBuildsTheSidecarCommand(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "/ambient/lib")
	reviewer, err := NewBundledReviewer(BundledReviewerConfig{
		Python:      "/venv/bin/python",
		SidecarPath: "/cache/sidecar",
		ModelsRoot:  "/data/image-tools",
		Libraries:   []string{"/venv/nvidia/cu13/lib", "/venv/nvidia/cudnn/lib"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reviewer.program != "/venv/bin/python" {
		t.Fatalf("program = %q; the reviewer must run from the private venv, never a host Python", reviewer.program)
	}
	wantArgs := []string{"-m", "image_tools_sidecar.minor_review", "--models-root", "/data/image-tools"}
	if !slices.Equal(reviewer.args, wantArgs) {
		t.Fatalf("args = %v, want %v", reviewer.args, wantArgs)
	}
	for _, want := range []string{
		"PYTHONPATH=/cache/sidecar",
		"CUDA_VISIBLE_DEVICES=", // CPU only: a review never takes VRAM from a generation
		"IMAGE_TOOLS_DEVICE=cpu",
		"LD_LIBRARY_PATH=/venv/nvidia/cu13/lib:/venv/nvidia/cudnn/lib:/ambient/lib",
	} {
		if !slices.Contains(reviewer.env, want) {
			t.Errorf("env missing %q: %v", want, reviewer.env)
		}
	}
	if reviewer.timeout <= 0 {
		t.Fatal("the bundled reviewer must be bounded by a timeout (an expired review fails closed)")
	}
}

func TestNewBundledReviewerRequiresEveryLocation(t *testing.T) {
	full := BundledReviewerConfig{Python: "/p", SidecarPath: "/s", ModelsRoot: "/m"}
	for name, cfg := range map[string]BundledReviewerConfig{
		"python":  {SidecarPath: full.SidecarPath, ModelsRoot: full.ModelsRoot},
		"sidecar": {Python: full.Python, ModelsRoot: full.ModelsRoot},
		"models":  {Python: full.Python, SidecarPath: full.SidecarPath},
	} {
		if _, err := NewBundledReviewer(cfg); err == nil {
			t.Errorf("config without %s was accepted", name)
		}
	}
}

func TestBundledReviewerModelsAreTheRegistryIDsTheSidecarReads(t *testing.T) {
	// The profile is the single source for the classifier and escalation ids.
	profile := DefaultReviewProfile(TierLocal)
	if want := []string{"falconsai-nsfw", "adamcodd-vit-nsfw"}; !slices.Equal(profile.ClassifierIDs(), want) {
		t.Fatalf("classifier ids = %v, want %v", profile.ClassifierIDs(), want)
	}
	if want := []string{"yunet", "fairface-vit-age"}; !slices.Equal(profile.EscalationIDs(), want) {
		t.Fatalf("escalation ids = %v, want %v", profile.EscalationIDs(), want)
	}
}

func TestReviewerExitCodeFourNamingWeightsIsAMissingWeightsError(t *testing.T) {
	t.Setenv("GO_WANT_MINOR_HINTS", "missing")
	reviewer, err := NewCommandReviewer(os.Args[0] + " -test.run=TestMinorReviewerHintsHelperProcess")
	if err != nil {
		t.Fatal(err)
	}
	_, err = reviewer.Review(context.Background(), [][]byte{[]byte("x")})
	var missing *MissingWeightsError
	if !errors.As(err, &missing) || !slices.Equal(missing.IDs, []string{"yunet", "fairface-vit-age"}) {
		t.Fatalf("err = %v, want a MissingWeightsError naming yunet and fairface-vit-age", err)
	}
}

func TestReviewerExitCodeFourWithoutNamesIsAnOrdinaryFailure(t *testing.T) {
	t.Setenv("GO_WANT_MINOR_HINTS", "exit4")
	reviewer, err := NewCommandReviewer(os.Args[0] + " -test.run=TestMinorReviewerHintsHelperProcess")
	if err != nil {
		t.Fatal(err)
	}
	_, err = reviewer.Review(context.Background(), [][]byte{[]byte("x")})
	var missing *MissingWeightsError
	if err == nil || errors.As(err, &missing) {
		t.Fatalf("err = %v, want a plain failure (nothing to acquire)", err)
	}
}

func TestUnavailableReviewerFailsClosedWithTheCause(t *testing.T) {
	_, err := UnavailableReviewer(errors.New("sidecar missing")).Review(context.Background(), [][]byte{[]byte("x")})
	if err == nil || !strings.Contains(err.Error(), "sidecar missing") {
		t.Fatalf("err = %v; an unbuildable reviewer must refuse, naming why", err)
	}
}

func TestMinorVerdictSensitive(t *testing.T) {
	sexual, top := MinorVerdict{Images: []MinorImageFinding{{NSFWScore: 0.2}, {NSFWScore: 0.91, Sexual: true}}}.Sensitive()
	if !sexual || top != 0.91 {
		t.Fatalf("sensitive = %v, top = %v", sexual, top)
	}
	if sexual, _ := (MinorVerdict{Images: []MinorImageFinding{{NSFWScore: 0.01}}}).Sensitive(); sexual {
		t.Fatal("a clear image was reported sensitive")
	}
	// An operator-supplied reviewer reports only the aggregate flag.
	if sexual, _ := (MinorVerdict{Sexual: true}).Sensitive(); !sexual {
		t.Fatal("aggregate sexual flag was dropped")
	}
}

func TestCommandReviewerSendsStrictAndAssumeSexualHintsOnlyWhenAsked(t *testing.T) {
	t.Setenv("GO_WANT_MINOR_HINTS", "1")
	reviewer, err := NewCommandReviewer(os.Args[0] + " -test.run=TestMinorReviewerHintsHelperProcess")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		ctx  context.Context
		want string
	}{
		"none":   {context.Background(), "strict=false assume=false"},
		"strict": {WithStrictReview(context.Background()), "strict=true assume=false"},
		"both":   {WithAssumeSexual(WithStrictReview(context.Background())), "strict=true assume=true"},
	}
	for name, tc := range cases {
		verdict, err := reviewer.Review(tc.ctx, [][]byte{[]byte("\x89PNG\r\n\x1a\nrest")})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if verdict.Detail != tc.want {
			t.Errorf("%s: request carried %q, want %q", name, verdict.Detail, tc.want)
		}
	}
}

func TestCommandReviewerAppliesItsEnvironmentToTheSubprocess(t *testing.T) {
	t.Setenv("GO_WANT_MINOR_HINTS", "env")
	reviewer, err := NewCommandReviewer(os.Args[0] + " -test.run=TestMinorReviewerHintsHelperProcess")
	if err != nil {
		t.Fatal(err)
	}
	reviewer.env = []string{"MINOR_REVIEWER_PROBE=from-config"}
	verdict, err := reviewer.Review(context.Background(), [][]byte{[]byte("x")})
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Detail != "from-config" {
		t.Fatalf("subprocess saw %q", verdict.Detail)
	}
}

// TestMinorReviewerHintsHelperProcess reports which request hints it received (or
// an environment probe) in the verdict detail.
func TestMinorReviewerHintsHelperProcess(t *testing.T) {
	mode := os.Getenv("GO_WANT_MINOR_HINTS")
	if mode == "" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	var request minorReviewRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		os.Exit(3)
	}
	switch mode {
	case "missing":
		fmt.Fprint(os.Stderr, "minor_review: model weights are not installed: missing-weights=yunet,fairface-vit-age")
		os.Exit(4)
	case "exit4":
		fmt.Fprint(os.Stderr, "something else exited 4")
		os.Exit(4)
	}
	detail := fmt.Sprintf("strict=%t assume=%t", request.Strict, request.AssumeSexual)
	if mode == "env" {
		detail = os.Getenv("MINOR_REVIEWER_PROBE")
	}
	out, _ := json.Marshal(MinorVerdict{Detail: detail})
	_, _ = os.Stdout.Write(out)
	os.Exit(0)
}
