package runtimeapp

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/artifactlease"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/cloudtarget"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/gocache"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/hostwatchdog"
	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/peerrecord"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/projectstate"
	"github.com/vrooli/vrooli/internal/recovery"
	"github.com/vrooli/vrooli/internal/releaseauthority"
	"github.com/vrooli/vrooli/internal/resources"
	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/resources/wordpress"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/safeguards/emergency-watchdog"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/setpoint"
	"github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/setup/assets"
	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/templatevalidation"
	"github.com/vrooli/vrooli/internal/tools/triposr-source"
	"github.com/vrooli/vrooli/packages/kopiaregistry-go"
	"github.com/vrooli/vrooli/packages/recoverypoint"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// versionedStateStores is the migration registry for owners whose stored format
// is declared as a version rather than inferred from their source. An owner
// that is not listed here is still emitted as an opaque identity store, which
// keeps the old "any change is breaking" envelope until it is converted.
//
// Converting an owner means: declare its write version and read versions, derive
// its schema digest from its compiled types, declare the state paths a forced
// transition must snapshot, and prove any additive claim with a downlevel
// fixture. The opaque entry then disappears without a second code path.
var versionedStateStores = map[string]func() (stateformat.Store, error){
	"runtime-registry":     func() (stateformat.Store, error) { return scenarioruntime.StateFormat(), nil },
	"core-activation":      coreactivation.StateFormat,
	"capacity":             capacity.StateFormat,
	"host-session":         hostsession.StateFormat,
	"process-records":      process.StateFormat,
	"operator-capability":  operatorcapability.StateFormat,
	"host-lighting-intent": lightingruntime.StateFormat,
	// The progress record's paths are owned here, beside its only writer.
	"root-rebaseline-progress":    rebaselineProgressStateFormat,
	"project-setup":               projectstate.StateFormat,
	"setup-state":                 setup.StateFormat,
	"sustain-observations":        setpoint.StateFormat,
	"installation-inventory":      cliinstall.StateFormat,
	"peer-records":                peerrecord.StateFormat,
	"watchdog-publication":        emergencywatchdog.StateFormat,
	"agent-sessions":              agentsession.StateFormat,
	"scenario-contract":           scenario.StateFormat,
	"resource-manifest":           manifest.StateFormat,
	"setup-assets":                assets.StateFormat,
	"triposr-source":              triposrsource.StateFormat,
	"coding-agent-recovery":       recovery.StateFormat,
	"operator-state":              operatorstate.StateFormat,
	"supervisor-owner":            runtimesupervisor.StateFormat,
	"artifact-leases":             artifactlease.StateFormat,
	"secure-store":                securestore.StateFormat,
	"agent-catalogs":              agentCatalogsStateFormat,
	"resource-artifact-cache":     resourcedeployment.StateFormat,
	"template-validation-markers": templatevalidation.StateFormat,
	"scenario-cli-install":        scenarioCLIInstallStateFormat,
	"release-authority":           releaseauthority.StateFormat,
	"baseline-floor":              baselinefloor.StateFormat,
	"retained-runtime":            retainedruntime.StateFormat,
	"kopia-registry":              kopiaregistry.StateFormat,
	"recovery-points":             recoverypoint.StateFormat,
	"guard-modes":                 guardModesStateFormat,
	"go-cache":                    gocache.StateFormat,
	"tool-installation":           toolInstallationStateFormat,
	"break-glass":                 breakGlassStateFormat,
	"credential-recovery":         credentialRecoveryStateFormat,
	"harness-lifecycle":           harnessLifecycleStateFormat,
	"prepared-root-bundle":        preparedRootBundleStateFormat,
	"host-watchdog-state":         hostwatchdog.StateFormat,
	"companion-state":             resources.CompanionStateFormat,
	"cloud-target":                cloudtarget.StateFormat,
	"wordpress-state":             wordpress.StateFormat,
	// Derived caches: real bytes, no compatibility contract. See
	// stateformat_discardable.go for the reader evidence behind each claim.
	"lifecycle-cache":  lifecycleCacheStateFormat,
	"host-facts-cache": hostFactsCacheStateFormat,
}

// NativeStateFormatManifest is the versioned compatibility manifest for this
// executable. Stores that have a declared format are emitted from it; every
// other owner is emitted as an opaque identity store until it is migrated.
// Opaque stores keep the old safety envelope: any change is breaking.
func NativeStateFormatManifest() (stateformat.Manifest, error) {
	base, err := NativePersistentStateManifest()
	if err != nil {
		return stateformat.Manifest{}, err
	}
	stores := make([]stateformat.Store, 0, len(base.Owners))
	converted := map[string]bool{}
	for _, owner := range base.Owners {
		if provider, versioned := versionedStateStores[owner.Owner]; versioned {
			store, err := provider()
			if err != nil {
				return stateformat.Manifest{}, fmt.Errorf("state format for owner %s: %w", owner.Owner, err)
			}
			stores = append(stores, store)
			converted[owner.Owner] = true
			continue
		}
		store, err := stateformat.Opaque(owner.Owner, owner.Digest)
		if err != nil {
			return stateformat.Manifest{}, fmt.Errorf("state format for owner %s: %w", owner.Owner, err)
		}
		stores = append(stores, store)
	}
	for name := range versionedStateStores {
		if !converted[name] {
			return stateformat.Manifest{}, fmt.Errorf("versioned state store %s has no persistence owner", name)
		}
	}
	if err := stateformat.SortStores(stores); err != nil {
		return stateformat.Manifest{}, err
	}
	manifest := stateformat.Manifest{Format: stateformat.ManifestFormat, Stores: stores}
	if err := manifest.Seal(); err != nil {
		return stateformat.Manifest{}, err
	}
	return manifest, nil
}
