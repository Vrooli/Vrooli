package cliutil

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type credentialContextRoundTripper func(*http.Request) (*http.Response, error)

func (f credentialContextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIdentityForwardingTransportMarksOnlyResolvedExactLocalPair(t *testing.T) {
	const baseline = "baseline-proof"
	const operation = "operation-proof"
	startCredentialContextBroker(t, baseline, operation)
	t.Setenv(EnvIdentityToken, baseline)

	var exactPair bool
	transport := identityForwardingTransport{next: credentialContextRoundTripper(func(req *http.Request) (*http.Response, error) {
		exactPair = HasResolvedAgentCredentialPair(req)
		return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Header: make(http.Header), Request: req}, nil
	})}
	request := httptestRequest("http://localhost/api")
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if !exactPair {
		t.Fatal("successfully broker-resolved local pair was not marked for API forwarding")
	}

	request = httptestRequest("http://localhost/api")
	request.Header.Set(HeaderAgentIdentityToken, "foreign-baseline")
	request.Header.Set(HeaderAgentOperationToken, "foreign-operation")
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if exactPair {
		t.Fatal("explicit foreign request identity received a broker-pair marker")
	}
}

func TestResolvedCredentialPairMarkerRejectsMutationRemoteAndRequiresPinnedResolution(t *testing.T) {
	const baseline = "pinned-baseline"
	const operation = "pinned-operation"
	startCredentialContextBroker(t, baseline, operation)
	t.Setenv(EnvIdentityToken, baseline)

	var marked bool
	var forwarded *http.Request
	transport := identityForwardingTransport{next: credentialContextRoundTripper(func(req *http.Request) (*http.Response, error) {
		marked = HasResolvedAgentCredentialPair(req)
		forwarded = req
		return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Header: make(http.Header), Request: req}, nil
	})}
	pin := &agentCredentialPin{
		token: baseline, operationToken: operation,
		resolvePair: func(context.Context) (AgentCredentials, error) {
			return AgentCredentials{BaselineToken: baseline, OperationToken: operation}, nil
		},
	}
	request := httptestRequest("http://localhost/api")
	request = request.WithContext(context.WithValue(request.Context(), agentCredentialPinKey{}, pin))
	request.Header.Set(HeaderAgentIdentityToken, baseline)
	request.Header.Set(HeaderAgentOperationToken, operation)
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if !marked {
		t.Fatal("successfully validated pinned pair was not marked")
	}

	request = httptestRequest("http://localhost/api")
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if !marked {
		t.Fatal("broker pair was not marked before mutation check")
	}
	forwarded.Header.Add(HeaderAgentOperationToken, "duplicate-operation")
	if HasResolvedAgentCredentialPair(forwarded) {
		t.Fatal("duplicate operation header retained resolved-pair status")
	}
	forwarded.Header.Set(HeaderAgentOperationToken, "changed-operation")
	if HasResolvedAgentCredentialPair(forwarded) {
		t.Fatal("mutated operation header retained resolved-pair status")
	}

	remote := httptestRequest("https://remote.invalid/api")
	remote.Header.Set(HeaderAgentIdentityToken, baseline)
	remote.Header.Set(HeaderAgentOperationToken, operation)
	if _, err := transport.RoundTrip(remote); err != nil {
		t.Fatal(err)
	}
	if marked || HasResolvedAgentCredentialPair(forwarded) || forwarded.Header.Get(HeaderAgentIdentityToken) != "" || forwarded.Header.Get(HeaderAgentOperationToken) != "" {
		t.Fatal("remote request retained identity or received resolved-pair status")
	}
}

func httptestRequest(url string) *http.Request {
	request, _ := http.NewRequest(http.MethodGet, url, nil)
	return request
}

func startCredentialContextBroker(t *testing.T, baseline, operation string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("/tmp", "cli-context-cred-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmpDir)
	if err := os.MkdirAll(filepath.Dir(AgentCredentialSocket()), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", AgentCredentialSocket())
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"baseline_token": baseline, "operation_token": operation})
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = os.RemoveAll(tmpDir)
	})
}
