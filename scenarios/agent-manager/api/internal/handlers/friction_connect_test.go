package handlers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type stubFrictionBoard struct {
	err   error
	since time.Time
}

func (b *stubFrictionBoard) ReportFriction(_ context.Context, fingerprint string, since, _ time.Time) (usageattribution.FrictionItem, error) {
	b.since = since
	return usageattribution.FrictionItem{Fingerprint: fingerprint, Status: usageattribution.StatusReported}, b.err
}

func (b *stubFrictionBoard) DismissFriction(_ context.Context, _ string, since, _ time.Time) error {
	b.since = since
	return b.err
}

// Report and dismiss act in the window the caller listed the finding in and
// say why they failed: an unknown fingerprint is NotFound, an unreachable bug
// inbox Unavailable, a malformed window InvalidArgument, anything else
// Internal.
func TestFrictionActionsCarryTheWindowAndMapErrors(t *testing.T) {
	board := &stubFrictionBoard{}
	h := &Handler{frictionBoard: board}
	since := time.Now().UTC().Add(-30 * 24 * time.Hour).Truncate(time.Second)
	report := func(since string) error {
		_, err := h.ReportFriction(context.Background(), connect.NewRequest(&domainpb.ReportFrictionRequest{Fingerprint: "fp", Since: since}))
		return err
	}
	dismiss := func(since string) error {
		_, err := h.DismissFriction(context.Background(), connect.NewRequest(&domainpb.DismissFrictionRequest{Fingerprint: "fp", Since: since}))
		return err
	}
	for name, call := range map[string]func(string) error{"report": report, "dismiss": dismiss} {
		board.err = nil
		if err := call(since.Format(time.RFC3339)); err != nil || !board.since.Equal(since) {
			t.Fatalf("%s: err %v, window %v; want the listed window %v", name, err, board.since, since)
		}
		if err := call(""); err != nil || time.Since(board.since) < 6*24*time.Hour {
			t.Fatalf("%s: empty since read %v, want the default week", name, board.since)
		}
		if err := call("last tuesday"); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("%s: malformed since = %v", name, err)
		}
		for err, want := range map[error]connect.Code{
			fmt.Errorf("%w %q", usageattribution.ErrUnknownFinding, "fp"):                 connect.CodeNotFound,
			fmt.Errorf("%w: connection refused", usageattribution.ErrFrictionUnavailable): connect.CodeUnavailable,
			errors.New("database disk image is malformed"):                                connect.CodeInternal,
		} {
			board.err = err
			if got := connect.CodeOf(call("")); got != want {
				t.Errorf("%s: %v mapped to %v, want %v", name, err, got, want)
			}
		}
	}
}
