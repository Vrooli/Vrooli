package inventory

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
	"google.golang.org/protobuf/proto"
)

func TestScanReturnsOneKeyedFindingPerSubject(t *testing.T) {
	h := NewConnectHandler(Deps{})
	request := connect.NewRequest(&inventoryv1.ScanRequest{Subjects: []*inventoryv1.Subject{
		{Kind: "asset", Id: "react-component-library:Button", Version: "1.2.0", Fingerprint: "sha256:abc"},
		{Kind: "asset", Id: "react-component-library:Card", Version: "1.0.0", Fingerprint: "sha256:def"},
	}})
	response, err := h.Scan(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetFindings()) != 2 {
		t.Fatalf("findings = %d, want 2", len(response.Msg.GetFindings()))
	}
	for i, finding := range response.Msg.GetFindings() {
		if !proto.Equal(finding.GetSubject(), request.Msg.GetSubjects()[i]) {
			t.Fatalf("finding %d subject = %v, want %v", i, finding.GetSubject(), request.Msg.GetSubjects()[i])
		}
	}
}

func TestScanRejectsSubjectWithoutID(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.Scan(context.Background(), connect.NewRequest(&inventoryv1.ScanRequest{Subjects: []*inventoryv1.Subject{{Kind: "asset"}}}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error = %v, want invalid argument", err)
	}
}
