package validationbroker

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

type frozenReplayScenarioResolver struct {
	url string
}

func (r frozenReplayScenarioResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type frozenReplayRoundTripper func(*http.Request) (*http.Response, error)

func (f frozenReplayRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestLiveFrozenReplayTestRunnerUsesOwnedDependencies(t *testing.T) {
	wantErr := errors.New("injected transport reached")
	var gotURL string
	client := &http.Client{Transport: frozenReplayRoundTripper(func(request *http.Request) (*http.Response, error) {
		gotURL = request.URL.String()
		return nil, wantErr
	})}
	runner := LiveFrozenReplayTestRunner{
		Resolver:   frozenReplayScenarioResolver{url: "http://workspace-sandbox.test"},
		HTTPClient: client,
	}
	_, err := runner.Start(context.Background(), "idempotency-key", &validationv1.FrozenReplayExecution{CaseId: "case-1"}, FrozenEvaluatorBundle{}, time.Second)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Start error = %v, want injected transport error", err)
	}
	if !strings.HasPrefix(gotURL, "http://workspace-sandbox.test/") {
		t.Fatalf("request URL = %q, want injected workspace-sandbox URL", gotURL)
	}
}
