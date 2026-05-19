// Package replay_config hosts the BAS ReplayConfigService Connect-RPC handler.
//
// ReplayConfigService owns the single persisted replay-style configuration
// (chrome theme, cursor, background, watermark, intro/outro card, browser
// scale, etc.). The payload is intentionally free-form (google.protobuf.Struct);
// server-side export helpers in handlers/replay_config.go interpret known
// keys and tolerate unknown ones.
package replay_config

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	replayconfigconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config/replay_configconnect"
)

// SettingsStore is the narrow seam for persisting the single replay-config
// blob. Production wiring uses the database repository.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	DeleteSetting(ctx context.Context, key string) error
}

// Deps wires the replay_config handler.
type Deps struct {
	// Store persists the replay configuration blob. Required.
	Store SettingsStore
	// Logger is required.
	Logger *logrus.Logger
}

// Module builds the ReplayConfigService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("replay_config.Module requires Deps.Logger")
	}
	if d.Store == nil {
		panic("replay_config.Module requires Deps.Store")
	}
	path, handler := replayconfigconnect.NewReplayConfigServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ replayconfigconnect.ReplayConfigServiceHandler = (*service)(nil)
