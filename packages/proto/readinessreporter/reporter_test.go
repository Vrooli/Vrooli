package readinessreporter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
)

type readinessServiceStub struct {
	readinessconnect.UnimplementedReadinessServiceHandler
	calls int
	auth  string
}

func (s *readinessServiceStub) ReportEvidence(_ context.Context, request *connect.Request[readinessv1.ReportEvidenceRequest]) (*connect.Response[readinessv1.ReportEvidenceResponse], error) {
	s.calls++
	s.auth = request.Header().Get("Authorization")
	return connect.NewResponse(&readinessv1.ReportEvidenceResponse{}), nil
}

func readinessServer(t *testing.T) (*httptest.Server, *readinessServiceStub) {
	t.Helper()
	stub := &readinessServiceStub{}
	_, handler := readinessconnect.NewReadinessServiceHandler(stub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, stub
}

func readinessIdentity() Identity {
	return Identity{
		Scenario: "offer-desk", ProfileID: "pilot", CandidateCommit: "commit", ArtifactDigest: "digest",
		Targets: []string{"linux"}, Channel: "local", PolicyVersion: 1,
	}
}

func TestReporterResolvesCurrentDeploymentManagerForEachReport(t *testing.T) {
	first, firstStub := readinessServer(t)
	second, secondStub := readinessServer(t)
	currentURL := first.URL
	resolveCalls := 0
	reporter, err := New(Config{
		ResolveBaseURL: func(context.Context) (string, error) {
			resolveCalls++
			return currentURL, nil
		},
		Token: "readiness-token", Identity: readinessIdentity(), CriterionID: "catalog-ready",
		ProducerBinding: "offer-desk.catalog", ProducerVersion: "offer-desk",
	})
	if err != nil {
		t.Fatal(err)
	}

	report := func() error {
		return reporter.ReportStatus(context.Background(), "offer-desk", "passed", "catalog reconciled", "offer-desk:readiness", time.Now().UTC())
	}
	if err := report(); err != nil {
		t.Fatal(err)
	}
	first.Close()
	currentURL = second.URL
	if err := report(); err != nil {
		t.Fatalf("a peer restart must not leave the reporter pinned to its original address: %v", err)
	}

	if resolveCalls != 2 || firstStub.calls != 1 || secondStub.calls != 1 || secondStub.auth != "Bearer readiness-token" {
		t.Fatalf("resolve calls=%d first calls=%d second calls=%d second auth=%q", resolveCalls, firstStub.calls, secondStub.calls, secondStub.auth)
	}
}

func TestReporterKeepsFixedBaseURLModeForPinnedDeployments(t *testing.T) {
	server, stub := readinessServer(t)
	reporter, err := New(Config{
		BaseURL: server.URL, Token: "readiness-token", Identity: readinessIdentity(), CriterionID: "catalog-ready",
		ProducerBinding: "offer-desk.catalog", ProducerVersion: "offer-desk",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := reporter.ReportStatus(context.Background(), "offer-desk", "passed", "ready", "fixed-address", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if stub.calls != 1 || stub.auth != "Bearer readiness-token" {
		t.Fatalf("calls=%d auth=%q", stub.calls, stub.auth)
	}
}

func TestReporterRequiresExactlyOneAddressMode(t *testing.T) {
	base := Config{Token: "token", Identity: readinessIdentity(), CriterionID: "criterion", ProducerBinding: "binding", ProducerVersion: "producer"}
	_, err := New(base)
	if err == nil || !strings.Contains(err.Error(), "requires a URL") {
		t.Fatalf("error = %v; want missing address error", err)
	}
	base.BaseURL = "http://127.0.0.1:1"
	base.ResolveBaseURL = func(context.Context) (string, error) { return "http://127.0.0.1:2", nil }
	_, err = New(base)
	if err == nil || !strings.Contains(err.Error(), "either a fixed URL or a URL resolver") {
		t.Fatalf("error = %v; want mutually exclusive address modes", err)
	}
}

func TestReporterRejectsEmptyResolvedAddress(t *testing.T) {
	reporter, err := New(Config{
		ResolveBaseURL: func(context.Context) (string, error) { return " ", nil },
		Token:          "token", Identity: readinessIdentity(), CriterionID: "criterion", ProducerBinding: "binding", ProducerVersion: "producer",
		HTTPClient: &http.Client{},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = reporter.ReportStatus(context.Background(), "offer-desk", "passed", "ready", "ref", time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "empty URL") {
		t.Fatalf("error = %v; want empty URL error", err)
	}
}
