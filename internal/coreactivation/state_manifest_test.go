package coreactivation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestServingManifestProbeFailureDoesNotClaimLegacy(t *testing.T) {
	for _, failure := range []error{errors.New("permission denied"), context.Canceled, errors.New("invalid state manifest response")} {
		got := explainStateContractChange(context.Background(), "serving", "candidate", func(_ context.Context, image string) (statecontract.Manifest, error) {
			if image == "serving" {
				return statecontract.Manifest{}, failure
			}
			return statecontract.Manifest{}, nil
		})
		if len(got) != 1 || !strings.Contains(got[0], failure.Error()) || strings.Contains(got[0], "predates") || strings.Contains(got[0], "always differs") {
			t.Fatalf("misclassified %v: %v", failure, got)
		}
	}
}
