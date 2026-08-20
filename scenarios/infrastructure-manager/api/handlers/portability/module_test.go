package portability

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	internalportability "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
)

func handlerFor(root string) *connectHandler {
	return &connectHandler{service: internalportability.NewService(root, nil)}
}

// repoRoot walks up to the repository root, identified by the capability
// vocabulary the grid is computed against.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli", "capability-vocabulary.json")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no repository root with .vrooli/capability-vocabulary.json was found above the test directory")
		}
		dir = parent
	}
}

// TestUnusableRootIsFailedPreconditionNotAnEmptyGrid pins the contract that an
// unreadable manifest tree is a failure, never a grid with no rows. An empty
// grid returned OK reads as "this repository declares no capabilities", which
// is a claim about the repository rather than a report about the read.
func TestUnusableRootIsFailedPreconditionNotAnEmptyGrid(t *testing.T) {
	handler := handlerFor(t.TempDir())
	_, err := handler.GetGrid(context.Background(), connect.NewRequest(&portabilityv1.GetGridRequest{}))
	if err == nil {
		t.Fatal("GetGrid returned a grid for an unusable manifest root")
	}
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("GetGrid returned code %v, want FailedPrecondition", got)
	}
	if !strings.Contains(err.Error(), "capability-vocabulary.json") {
		t.Errorf("error %q does not name the file whose absence was decisive", err)
	}

	if _, err := handler.GetFleet(context.Background(), connect.NewRequest(&portabilityv1.GetFleetRequest{})); err == nil {
		t.Fatal("GetFleet returned a readout for an unusable manifest root")
	} else if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("GetFleet returned code %v, want FailedPrecondition", got)
	}
}

func TestGetCapabilityRequiresAName(t *testing.T) {
	handler := handlerFor(t.TempDir())
	_, err := handler.GetCapability(context.Background(), connect.NewRequest(&portabilityv1.GetCapabilityRequest{}))
	if err == nil {
		t.Fatal("GetCapability accepted an empty capability name")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("GetCapability returned code %v, want InvalidArgument", got)
	}
}

// TestEnumProjectionsCoverEveryToken guards the failure mode the projection
// helpers exist for: a token that resolves to no enum value renders as
// UNSPECIFIED, and an unlabelled value is indistinguishable from a deliberate
// one. Every token the domain can emit must land on a labelled enum value.
func TestEnumProjectionsCoverEveryToken(t *testing.T) {
	grid, err := handlerFor(repoRoot(t)).service.Grid(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range grid.Capabilities {
		if protoSituation(entry.Situation) == portabilityv1.CapabilitySituation_CAPABILITY_SITUATION_UNSPECIFIED {
			t.Errorf("situation %q projects to UNSPECIFIED", entry.Situation)
		}
		for _, platform := range entry.Platforms {
			if protoHostOS(platform.HostOS) == portabilityv1.HostOS_HOST_OS_UNSPECIFIED {
				t.Errorf("host OS %q projects to UNSPECIFIED", platform.HostOS)
			}
			if protoStatus(platform.Status) == portabilityv1.ResolutionStatus_RESOLUTION_STATUS_UNSPECIFIED {
				t.Errorf("resolution status %q projects to UNSPECIFIED", platform.Status)
			}
			if protoQualification(platform.Qualification) == portabilityv1.Qualification_QUALIFICATION_UNSPECIFIED {
				t.Errorf("qualification %q projects to UNSPECIFIED", platform.Qualification)
			}
		}
	}
}

func TestTierProjectionStripsTheOrdinalPrefix(t *testing.T) {
	readout, err := handlerFor(repoRoot(t)).service.Fleet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(readout.TierUpgrades) == 0 {
		t.Skip("the live repository reports no tier upgrade candidates")
	}
	fleet := protoFleet(readout)
	for _, upgrade := range fleet.GetTierUpgrades() {
		if upgrade.GetCurrentTier() == portabilityv1.DeliveryTier_DELIVERY_TIER_UNSPECIFIED {
			t.Errorf("%s current tier projects to UNSPECIFIED", upgrade.GetScenario())
		}
		if upgrade.GetNextTier() == portabilityv1.DeliveryTier_DELIVERY_TIER_UNSPECIFIED {
			t.Errorf("%s next tier projects to UNSPECIFIED", upgrade.GetScenario())
		}
	}
}
