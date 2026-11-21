package terminal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"web-console/session"
	termcore "web-console/terminal"
)

type emptySessionManager struct{}

func (emptySessionManager) Get(string) (*session.Session, bool) { return nil, false }

func TestAdapterRejectsMissingSessionsAndMapsHelpers(t *testing.T) {
	a := &Adapter{Manager: emptySessionManager{}}
	ctx := context.Background()
	if _, err := a.GetScreen(ctx, "missing", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetScreen error = %v", err)
	}
	if _, err := a.SendInput(ctx, "missing", InputRequest{Variant: InputVariantText, Text: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SendInput error = %v", err)
	}
	if _, err := a.WaitIdle(ctx, "missing", 0, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("WaitIdle error = %v", err)
	}
	sgr := sgrToHandler(termcore.SGR{FG: 1, BG: 2, Bold: true, Italic: true, Underline: true, Inverse: true, Faint: true})
	if !sgr.Bold || sgr.FG != 1 || sgr.BG != 2 {
		t.Fatalf("sgr mapping = %#v", sgr)
	}
	if isUnknownKeyError(nil) || isUnknownKeyError(errors.New("other")) || !isUnknownKeyError(fmt.Errorf("%w FOO", session.ErrUnknownKey)) {
		t.Fatal("unrecognized-key classifier mismatch")
	}
}
