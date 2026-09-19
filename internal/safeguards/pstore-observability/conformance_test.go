package pstoreobservability

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler {
		return NewHandler(hostreqkit.SafeguardManifest{Name: "pstore_observability"})
	}, Name: "pstore_observability", Kind: hostreqspec.KindSafeguard, Checks: []string{"name_and_kind"}})
}
