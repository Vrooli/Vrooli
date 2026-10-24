package settings

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings"
)

type fakeSettingsService struct{ err error }

func (fakeSettingsService) GetDefaults() Defaults {
	return Defaults{DefaultBackend: "standard", DefaultPolicy: Policy{Mode: "ttl", Duration: "1h"}}
}

func (f fakeSettingsService) UpdateDefaults(UpdateDefaultsRequest) (Defaults, error) {
	return f.GetDefaults(), f.err
}

func TestConnectHandlerSettings(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeSettingsService{}})
	if resp, err := h.GetSessionDefaults(context.TODO(), connect.NewRequest(&settingsv1.GetSessionDefaultsRequest{})); err != nil || resp.Msg.Defaults.DefaultBackend != "standard" {
		t.Fatalf("get: %#v %v", resp, err)
	}
	backend := "tmux"
	if resp, err := h.UpdateSessionDefaults(context.TODO(), connect.NewRequest(&settingsv1.UpdateSessionDefaultsRequest{DefaultBackend: &backend, DefaultPolicy: &settingsv1.ExpirationPolicy{Mode: "never"}})); err != nil || resp.Msg.Defaults == nil {
		t.Fatalf("update: %#v %v", resp, err)
	}
	for _, in := range []error{ErrInvalidArgument, errors.New("db")} {
		h = NewConnectHandler(Deps{Service: fakeSettingsService{err: in}})
		_, err := h.UpdateSessionDefaults(context.TODO(), connect.NewRequest(&settingsv1.UpdateSessionDefaultsRequest{}))
		if err == nil {
			t.Fatalf("expected error for %v", in)
		}
	}
}
