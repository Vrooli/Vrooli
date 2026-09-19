package agentharness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliutil"
)

// EnforcementPosture describes the live enforcement caveats returned by
// resource-owned policy commands.
type EnforcementPosture struct {
	Permissions string   `json:"permissions"`
	Caveats     []string `json:"caveats,omitempty"`
}

type (
	ModelDiscoveryFunc      = agentcatalog.ModelDiscoveryFunc
	LiveModelCatalog        = agentcatalog.LiveModelCatalog
	CodingRoleCatalog       = agentcatalog.CodingRoleCatalog
	CodingRole              = agentcatalog.CodingRole
	ModelAlias              = agentcatalog.ModelAlias
	CatalogProvenance       = agentcatalog.CatalogProvenance
	PolicyValidationFinding = agentcatalog.PolicyValidationFinding
	CatalogFreshness        = agentcatalog.CatalogFreshness
)

var ErrModelDiscoveryUnavailable = agentcatalog.ErrModelDiscoveryUnavailable

func catalogStalenessFindings(c CodingRoleCatalog, now time.Time, againstLive bool) []PolicyValidationFinding {
	return agentcatalog.CatalogStalenessFindings(c, now, againstLive)
}

func liveCatalogFindings(c CodingRoleCatalog, live LiveModelCatalog) []PolicyValidationFinding {
	return agentcatalog.LiveCatalogFindings(c, live)
}

func parseObservedAt(value string) (time.Time, error) { return agentcatalog.ParseObservedAt(value) }

type CodingPolicyConfig struct {
	Runner      string
	CatalogPath string
	Posture     EnforcementPosture
	Stdout      io.Writer
	Stderr      io.Writer
	Discovery   ModelDiscoveryFunc
}

const CodingRolePolicySchemaVersion = agentcatalog.CodingRolePolicySchemaVersion

type PolicyValidationError struct {
	Code string
	Err  error
}

func (e *PolicyValidationError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *PolicyValidationError) Unwrap() error { return e.Err }

func policyFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func writeJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ResourceCatalogPath locates an installed resource catalog through the
// normal repository-root resolver.
func ResourceCatalogPath(resource string) string {
	return filepath.Join(cliutil.ResolveRepoRoot(), "resources", resource, "model-policy.json")
}
