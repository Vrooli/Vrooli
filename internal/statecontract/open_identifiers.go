package statecontract

import platform "github.com/vrooli/platform-go"

// OpenIdentifierFields are persisted identities, not a closed vocabulary.
// Comparing one with a particular identity selects a specific object; it does
// not define the set of values that the format accepts. This exception applies
// only to caller reads, never writes, and must name the declaring type exactly.
var OpenIdentifierFields = map[string]bool{
	// Scenario names and lifecycle step names are supplied by scenario
	// definitions. A special case for one agent's API step does not make all
	// scenario and step names a closed protocol vocabulary.
	"github.com/vrooli/vrooli/internal/process.Record.Scenario": true,
	"github.com/vrooli/vrooli/internal/process.Record.Step":     true,
	// Manifest names identify user-supplied resources; a special case does not close the set.
	"github.com/vrooli/vrooli/internal/resources/manifest.ResourceManifest.Name": true,
	// A release version identifies one artifact; it is not a closed enum.
	"github.com/vrooli/vrooli/internal/resources.ManagedServiceArtifact.Version": true,
}

// OpenDataFields carry user- or manifest-supplied values rather than a closed
// vocabulary, even when code selects a specific value for special handling.
var OpenDataFields = map[string]bool{
	"github.com/vrooli/vrooli/packages/resource-deployment.ServiceArtifact.Version": true,
	// An image reference is a manifest-supplied identity, not an enum.
	"github.com/vrooli/vrooli/internal/resources/manifest.ResourceRuntime.Image":      true,
	"github.com/vrooli/vrooli/internal/resources/manifest.ResourceHealthCheck.Target": true,
	"github.com/vrooli/vrooli/internal/resources.ComposeCandidateReceipt.Platform":    true,
	"github.com/vrooli/cli-core/agentcatalog.CodingRole.CanonicalModel":               true,
}

// StoredForeignValueFacts names exact cross-package bindings whose values are
// already explicit compiled facts of the field's persistence owner.
type ForeignValueFact struct {
	Owner, Fact, Value string
}

var StoredForeignValueFacts = map[string]ForeignValueFact{
	// resources.PersistentStateOwner includes native_launcher_publication_profile.
	"github.com/vrooli/vrooli/internal/resources.ManagedUpdateRuntime.NativeLauncherProfile|github.com/vrooli/platform-go.NativeLauncherProfileOpenRGB": {Owner: "resources", Fact: "native_launcher_publication_profile", Value: platform.NativeLauncherProfileOpenRGB},
}
