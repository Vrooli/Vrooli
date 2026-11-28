package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/packages/credential-authority-go/hubidentity"

	cleanupH "storage-manager/handlers/cleanup"
	"storage-manager/internal/modeleviction"
	"storage-manager/internal/workgate"
)

// startModelRetention runs the model-store retention controller and returns
// the decision effect the resolution webhook applies. Without the provider
// (policy unreadable at start) nothing runs and nothing is returned.
func startModelRetention(ctx context.Context, logger *log.Logger, service any, gate *workgate.Gate) []operatordecision.DomainEffect {
	provider, ok := cleanupH.ModelStoreRetention(service)
	if !ok {
		logger.Printf("model retention disabled: the model-store-retention provider is not registered")
		return nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		logger.Printf("model retention disabled: %v", err)
		return nil
	}
	store := modeleviction.NewFileStore(filepath.Join(configDir, "vrooli", "storage-manager", "model-eviction-asks.json"))
	notices := hubidentity.NewNoticeSender(modeleviction.Source)
	controller := &modeleviction.Controller{
		Decider: provider, Store: store, Notices: notices, Logf: logger.Printf,
		Raiser: operatordecision.Raiser{
			Source: modeleviction.Source, Kind: modeleviction.Kind, Store: store,
			Publisher: eventbus.NewDiscoveredClient(context.Background()), Window: modeleviction.AskWindow, Clock: schedule.System(),
		},
	}
	go controller.Run(ctx, gate.Run)
	return []operatordecision.DomainEffect{&modeleviction.Resolver{
		Decider: provider, Store: store, Notices: notices, Logf: logger.Printf,
		Async: func(fn func()) { go fn() },
	}}
}
