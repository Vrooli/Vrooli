package readinessreporter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	report := func() error {
		return reporter.ReportStatus(context.Background(), "offer-desk", "passed", "catalog reconciled", "offer-desk:readiness", time.Now().UTC())
	}
	require.NoError(t, report())
	first.Close()
	currentURL = second.URL
	require.NoError(t, report(), "a peer restart must not leave the reporter pinned to its original address")

	require.Equal(t, 2, resolveCalls)
	require.Equal(t, 1, firstStub.calls)
	require.Equal(t, 1, secondStub.calls)
	require.Equal(t, "Bearer readiness-token", secondStub.auth)
}

func TestReporterKeepsFixedBaseURLModeForPinnedDeployments(t *testing.T) {
	server, stub := readinessServer(t)
	reporter, err := New(Config{
		BaseURL: server.URL, Token: "readiness-token", Identity: readinessIdentity(), CriterionID: "catalog-ready",
		ProducerBinding: "offer-desk.catalog", ProducerVersion: "offer-desk",
	})
	require.NoError(t, err)
	require.NoError(t, reporter.ReportStatus(context.Background(), "offer-desk", "passed", "ready", "fixed-address", time.Now().UTC()))
	require.Equal(t, 1, stub.calls)
	require.Equal(t, "Bearer readiness-token", stub.auth)
}

func TestReporterRequiresExactlyOneAddressMode(t *testing.T) {
	base := Config{Token: "token", Identity: readinessIdentity(), CriterionID: "criterion", ProducerBinding: "binding", ProducerVersion: "producer"}
	_, err := New(base)
	require.ErrorContains(t, err, "requires URL")
	base.BaseURL = "http://127.0.0.1:1"
	base.ResolveBaseURL = func(context.Context) (string, error) { return "http://127.0.0.1:2", nil }
	_, err = New(base)
	require.ErrorContains(t, err, "either a fixed URL or a URL resolver")
}

func TestReporterRejectsEmptyResolvedAddress(t *testing.T) {
	reporter, err := New(Config{
		ResolveBaseURL: func(context.Context) (string, error) { return " ", nil },
		Token:          "token", Identity: readinessIdentity(), CriterionID: "criterion", ProducerBinding: "binding", ProducerVersion: "producer",
		HTTPClient: &http.Client{},
	})
	require.NoError(t, err)
	err = reporter.ReportStatus(context.Background(), "offer-desk", "passed", "ready", "ref", time.Now().UTC())
	require.ErrorContains(t, err, "empty URL")
}
