package autofiler

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"swarm-manager/internal/execution"
)

type fakeReviewClient struct {
	triggerErr error
	pollErr    error
	result     *execution.ReviewResult
	triggered  bool
	polled     bool
}

func (c *fakeReviewClient) TriggerReview(_ context.Context, _ execution.ReviewRequest) (string, error) {
	c.triggered = true
	if c.triggerErr != nil {
		return "", c.triggerErr
	}
	return "job-1", nil
}

func (c *fakeReviewClient) PollReview(_ context.Context, _ string) (*execution.ReviewResult, bool, error) {
	c.polled = true
	if c.pollErr != nil {
		return nil, false, c.pollErr
	}
	return c.result, true, nil
}

func TestFindingsFromGCTReview(t *testing.T) {
	findings := FindingsFromGCTReview("alpha", &execution.ReviewResult{Dimensions: []execution.ReviewDimension{
		{Name: "tests", Status: "red", Details: "2/10 passed"},
		{Name: "standards", Status: "yellow", Details: "3 warnings"},
		{Name: "docs", Status: "green"},
	}})
	if got, want := findingIDs(findings), []string{"gct:alpha:tests", "gct:alpha:standards"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("finding IDs = %#v, want %#v", got, want)
	}
	if findings[0].Severity != SeverityRed || findings[1].Severity != SeverityYellow {
		t.Fatalf("severities = %s/%s, want red/yellow", findings[0].Severity, findings[1].Severity)
	}
}

func TestGCTFindingSourcePollsAndMarksFreshness(t *testing.T) {
	store := NewReviewFreshnessStorePath(filepath.Join(t.TempDir(), "review_freshness.json"))
	client := &fakeReviewClient{result: &execution.ReviewResult{Dimensions: []execution.ReviewDimension{
		{Name: "tests", Status: "red", Details: "2/10 passed"},
	}}}
	source := GCTFindingSource{
		Client:        client,
		Freshness:     store,
		FreshnessTime: time.Hour,
		PollInterval:  time.Millisecond,
		Timeout:       time.Second,
	}
	findings, err := source.Findings(context.Background(), Target{Scenario: "alpha"})
	if err != nil {
		t.Fatalf("Findings: %v", err)
	}
	if len(findings) != 1 || findings[0].ID != "gct:alpha:tests" {
		t.Fatalf("findings = %#v, want gct alpha tests", findings)
	}
	if !client.triggered || !client.polled {
		t.Fatalf("triggered=%v polled=%v, want both true", client.triggered, client.polled)
	}

	secondClient := &fakeReviewClient{result: client.result}
	source.Client = secondClient
	second, err := source.Findings(context.Background(), Target{Scenario: "alpha"})
	if err != nil {
		t.Fatalf("Findings second: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second findings = %#v, want freshness skip", second)
	}
	if secondClient.triggered {
		t.Fatalf("fresh scenario should not trigger another review")
	}
}

func TestGCTFindingSourceDegradesOnDependencyErrors(t *testing.T) {
	source := GCTFindingSource{
		Client:  &fakeReviewClient{triggerErr: errors.New("gct stopped")},
		Timeout: time.Second,
	}
	findings, err := source.Findings(context.Background(), Target{Scenario: "alpha"})
	if err != nil {
		t.Fatalf("Findings error = %v, want nil", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want empty", findings)
	}
}

func findingIDs(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.ID)
	}
	return out
}
