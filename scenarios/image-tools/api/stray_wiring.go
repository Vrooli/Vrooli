package main

import (
	"context"
	"fmt"

	internaladapters "image-tools/internal/adapters"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/modelstray"
)

// newStrayScanner wires the containment scan to the live catalogs and install
// receipts: it sees seed and custom models and adapters, which of them have a
// receipt and where, and records a receipt in place when adopting files that
// already sit at a catalog entry's managed location.
func newStrayScanner(root string, models *internalmodels.Installer, adapters *internaladapters.Installer) (*modelstray.Scanner, error) {
	return modelstray.New(modelstray.Config{
		Root: root,
		Entries: func(ctx context.Context) ([]modelstray.Entry, error) {
			var out []modelstray.Entry
			catalog := models.Reg.Models()
			if models.Custom != nil {
				customs, err := models.Custom.List(ctx)
				if err != nil {
					return nil, err
				}
				catalog = append(catalog, customs...)
			}
			for _, m := range catalog {
				entry := modelstray.Entry{Kind: "model", ID: m.ID, Source: m.Source}
				if rec, ok, err := models.State.Get(ctx, m.ID); err == nil && ok && rec.Installed {
					entry.Installed, entry.Path = true, rec.Path
				}
				out = append(out, entry)
			}
			adapterCatalog := adapters.Reg.Adapters()
			if adapters.Custom != nil {
				customs, err := adapters.Custom.List(ctx)
				if err != nil {
					return nil, err
				}
				adapterCatalog = append(adapterCatalog, customs...)
			}
			for _, a := range adapterCatalog {
				entry := modelstray.Entry{Kind: "adapter", ID: a.ID, Source: a.Source}
				if rec, ok, err := adapters.State.Get(ctx, a.ID); err == nil && ok && rec.Installed {
					entry.Installed, entry.Path = true, rec.Path
				}
				out = append(out, entry)
			}
			return out, nil
		},
		Adopt: func(ctx context.Context, kind, id string) error {
			switch kind {
			case "model":
				_, err := models.AdoptExisting(ctx, id)
				return err
			case "adapter":
				_, err := adapters.AdoptExisting(ctx, id)
				return err
			}
			return fmt.Errorf("unknown catalog kind %q", kind)
		},
	})
}
