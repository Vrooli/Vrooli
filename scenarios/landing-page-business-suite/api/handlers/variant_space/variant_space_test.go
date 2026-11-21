package variant_space

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

func TestGetVariantSpacePreservesVerbatimJSON(t *testing.T) {
	payload := []byte(`{"_metadata":{"preserve":true}}`)
	response, err := NewHandler(func() []byte { return payload }).GetVariantSpace(context.Background(), connect.NewRequest(&lpbsv1.GetVariantSpaceRequest{}))
	if err != nil {
		t.Fatalf("GetVariantSpace() error = %v", err)
	}
	if got := string(response.Msg.GetRawJson()); got != string(payload) {
		t.Fatalf("raw JSON = %q, want %q", got, payload)
	}
}

func TestRegisterRoutesServesGeneratedConnectProcedure(t *testing.T) {
	router := mux.NewRouter()
	RegisterRoutes(router, func() []byte { return []byte(`{"_metadata":{"preserve":true}}`) })
	server := httptest.NewServer(router)
	defer server.Close()

	response, err := lpbsconnect.NewVariantSpaceServiceClient(server.Client(), server.URL).GetVariantSpace(context.Background(), connect.NewRequest(&lpbsv1.GetVariantSpaceRequest{}))
	if err != nil {
		t.Fatalf("generated Connect client GetVariantSpace() error = %v", err)
	}
	if got := string(response.Msg.GetRawJson()); got != `{"_metadata":{"preserve":true}}` {
		t.Fatalf("raw JSON = %q", got)
	}
}
