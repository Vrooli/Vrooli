package digesthttp

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

type fakeReader struct {
	days int32
}

func (f *fakeReader) GetBusinessDigest(_ context.Context, days int32) (*lpbsv1.BusinessDigest, error) {
	f.days = days
	return &lpbsv1.BusinessDigest{ContractVersion: "business-digest.v1"}, nil
}

func TestGetBusinessDigestValidatesWindowAndPreservesContract(t *testing.T) {
	reader := &fakeReader{}
	h := New(reader)
	response, err := h.GetBusinessDigest(context.Background(), connect.NewRequest(&lpbsv1.GetBusinessDigestRequest{WindowDays: 7}))
	if err != nil {
		t.Fatalf("expected valid digest request, got %v", err)
	}
	if reader.days != 7 || response.Msg.GetContractVersion() != "business-digest.v1" {
		t.Fatalf("unexpected digest response: days=%d contract=%q", reader.days, response.Msg.GetContractVersion())
	}
	if response.Msg.GetObservedAt() == nil {
		t.Fatal("expected producer observation timestamp")
	}
}

func TestGetBusinessDigestRejectsUnsupportedWindow(t *testing.T) {
	_, err := New(&fakeReader{}).GetBusinessDigest(context.Background(), connect.NewRequest(&lpbsv1.GetBusinessDigestRequest{WindowDays: 14}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid-argument error, got %v", err)
	}
}
