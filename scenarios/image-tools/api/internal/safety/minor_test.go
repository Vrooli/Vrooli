package safety

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
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
