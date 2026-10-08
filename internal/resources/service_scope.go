// Resource daemons must not inherit the cgroup of whoever started them: a
// resource restarted from a coding-agent session would otherwise live in that
// agent's scope and die with it (observed 2026-09-21: kokoro and whisper
// riding in a dead opencode session scope). Adoption into the services slice
// gives each resource process a scope of its own, sibling to scenario
// services, so its lifetime is tied to the operator's host, not the caller.
package resources

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	platform "github.com/vrooli/platform-go"
)

// resourceServiceScopeName returns the scope unit stem for one resource
// process. Placement and ownership inspection share this identity: the same
// resource/component tuple always maps to the same unit name. A digest
// preserves identity across sanitization and truncation, mirroring
// scenarioruntime.ServiceScopeName.
func resourceServiceScopeName(resource, component string) string {
	clean := func(value string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(value) {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				b.WriteRune(r)
			default:
				b.WriteRune('-')
			}
		}
		return strings.Trim(b.String(), "-")
	}
	name := "vrooli-service-resource-" + clean(resource) + "-" + clean(component)
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s%d:%s", len(resource), resource, len(component), component)))
	suffix := fmt.Sprintf("-%x", digest[:8])
	const limit = 200
	if len(name) > limit-len(suffix) {
		name = name[:limit-len(suffix)]
	}
	return strings.TrimRight(name, "-") + suffix
}

// placeResourceProcess adopts a resource-owned daemon into its own scope
// under the services slice. Failure is reported, not fatal: the process is
// already recorded and supervised, it just keeps running where it was born.
// Platforms without an adoption primitive are not an error.
func placeResourceProcess(resource, component string, pid int) error {
	_, _, err := platform.AdoptIntoScope(platform.AdoptSpec{
		PID:         pid,
		Scope:       resourceServiceScopeName(resource, component),
		Slice:       platform.ServicesSlice,
		Description: fmt.Sprintf("Vrooli resource %s (%s)", resource, component),
	})
	if err != nil && !errors.Is(err, platform.ErrUnsupported) {
		return err
	}
	return nil
}
