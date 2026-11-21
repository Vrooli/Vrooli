package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	securityhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1"
	securityhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/securityhealthv1connect"
)

type candidateScannerServiceStub struct {
	files map[string][]byte
	resp  *securityhealthv1.ScanCandidateResponse
}

func (s *candidateScannerServiceStub) ScanCandidate(_ context.Context, req *connect.Request[securityhealthv1.ScanCandidateRequest]) (*connect.Response[securityhealthv1.ScanCandidateResponse], error) {
	s.files = req.Msg.GetFiles()
	return connect.NewResponse(s.resp), nil
}

func TestCandidateSecurityClientForwardsExactIndexBlobsAndIncompleteStatus(t *testing.T) {
	files := map[string][]byte{
		"config with spaces.env": []byte("TOKEN=index-bytes\n"),
		"line\nbreak.txt":        []byte{0, 1, 2},
	}
	service := &candidateScannerServiceStub{resp: &securityhealthv1.ScanCandidateResponse{Complete: false, FailureCode: "scanner_unavailable"}}
	path, handler := securityhealthconnect.NewCandidateScannerServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := &CandidateSecurityClient{
		resolveURL: func(context.Context) (string, error) { return server.URL, nil },
		makeClient: func(baseURL string) securityhealthconnect.CandidateScannerServiceClient {
			return securityhealthconnect.NewCandidateScannerServiceClient(server.Client(), baseURL)
		},
	}
	result, err := client.Scan(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.FailureCode != "scanner_unavailable" {
		t.Fatalf("result = %+v", result)
	}
	if !reflect.DeepEqual(service.files, files) {
		t.Fatalf("received files = %#v, want exact index bytes %#v", service.files, files)
	}
}
