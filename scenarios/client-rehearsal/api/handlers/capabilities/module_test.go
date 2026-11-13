package capabilities

import (
	"context"
	"io"
	"log"
	"testing"

	"connectrpc.com/connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/shared"
)

func TestDescribeCapabilitiesFailsClosedWithoutRegistry(t *testing.T) {
	_, err := NewConnectHandler(nil, log.New(io.Discard, "", 0)).DescribeCapabilities(context.Background(), connect.NewRequest(&sharedv1.DescribeCapabilitiesRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Fatalf("code = %v, want unavailable", got)
	}
}

func TestModuleDeclaresConnectEndpoint(t *testing.T) {
	m := Module(nil, log.New(io.Discard, "", 0))
	if m.Name != "capabilities" {
		t.Fatalf("module name = %q", m.Name)
	}
	if len(m.Endpoints) != 1 || m.Endpoints[0].Path == "/api/v1/capabilities/describe" {
		t.Fatalf("expected generated Connect endpoint, got %#v", m.Endpoints)
	}
}
