package wire_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"research-desk/internal/home"
	"research-desk/internal/peer"
	"research-desk/internal/wire"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// A refusal's message says what is wrong without a command; the
// command travels beside it as a CommandHint the CLI prints.
func TestARefusalCarriesItsCommandBesideTheMessage(t *testing.T) {
	err := wire.Error(nil, "answer", home.WithCommand(home.Cause(home.ErrNotFound, fmt.Errorf("no topic %q", "x")), "research-desk topics"))
	var cerr *connect.Error
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeNotFound || cerr.Message() != `no topic "x"` {
		t.Fatalf("err = %v", err)
	}
	var hints []string
	for _, d := range cerr.Details() {
		if v, derr := d.Value(); derr == nil {
			if h, ok := v.(*sharedv1.CommandHint); ok {
				hints = append(hints, h.GetCommand())
			}
		}
	}
	if len(hints) != 1 || hints[0] != "research-desk topics" {
		t.Fatalf("hints = %v", hints)
	}

	plain := wire.Error(nil, "answer", home.Cause(home.ErrInvalid, errors.New("bad")))
	if errors.As(plain, &cerr); len(cerr.Details()) != 0 || strings.Contains(cerr.Message(), "research-desk") {
		t.Fatalf("an error without a command has no hint: %v", cerr.Details())
	}
}

// The access is an enum beside its phrase, so no client matches prose.
func TestTheWebConsoleAccessIsAKind(t *testing.T) {
	for said, want := range map[string]activityv1.WebConsoleAccess{
		peer.AccessCredential:   activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_SERVICE_CREDENTIAL,
		peer.AccessAvailability: activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY,
		"":                      activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_UNAVAILABLE,
	} {
		if got := wire.AccessKind(said); got != want {
			t.Errorf("AccessKind(%q) = %v, want %v", said, got, want)
		}
	}
}
