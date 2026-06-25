// Package portswitch is the deterministic primitive that switches a scenario's
// listener port between a canonical RANGE (dynamically allocated) and a free,
// conflict-free FIXED port inside the canonical band.
//
// It exists because tunnel-manager can only expose a scenario as a scenario
// route when that scenario declares a fixed UI port (the tunnel forwards to a
// concrete localhost:<port>); ranged scenarios had to be hand-edited. structure-
// health owns the port-band SSOT (internal/rules.CanonicalPortBand) and already
// performs format-preserving service.json edits (internal/svcedit), so the
// assign/release primitive belongs here, exposed over the ValidationService.
//
// Edits are format-preserving and the operations are idempotent and
// conflict-aware: assign scans every sibling scenario's fixed port (and,
// optionally, live listeners) and picks the lowest free in-band port; release
// reverts to the canonical range. Both support a dry-run (preview) so callers
// get the same before/after they would on apply.
package portswitch

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"structure-health/internal/intent"
	"structure-health/internal/rules"
	"structure-health/internal/svcedit"
)

// DefaultPortName is the port switched when a caller does not name one. UI is
// the port tunnel-manager forwards to, so it is the common case.
const DefaultPortName = "ui"

const serviceJSONRel = ".vrooli/service.json"

// Result reports the outcome of an assign/release, including a before/after so
// preview and apply share one shape (matching the FixConfig pattern).
type Result struct {
	Scenario     string
	PortName     string
	PreviousPort int    // the fixed port before the op (0 when ranged)
	AssignedPort int    // the fixed port after assign (0 after release)
	Changed      bool   // false when already in the target state (idempotent no-op)
	Applied      bool   // true when the edit was written to disk
	Before       string // service.json before
	After        string // service.json after
	Message      string
}

// Listening reports whether a TCP port is currently in use. Optional: when nil,
// assign considers only declared fixed ports across scenarios (still safe — the
// band reservation prevents collisions; live-listener avoidance is extra
// defence for ports allocated out-of-band).
type Listening func(port int) bool

// AssignFixed switches portName (default "ui") from a canonical range to a free
// in-band fixed port, format-preserving. Idempotent: a port that is already
// fixed is a no-op (Changed=false). When apply is false the edit is computed
// but not written.
func AssignFixed(scenarioRoot, portName string, apply bool, listening Listening) (Result, error) {
	portName = normalizePortName(portName)
	res := Result{Scenario: scenarioName(scenarioRoot), PortName: portName}

	in, err := intent.Load(scenarioRoot)
	if err != nil {
		return res, fmt.Errorf("load service.json for %q: %w", res.Scenario, err)
	}
	band, _, ok := rules.CanonicalPortBand(portName, "")
	if !ok {
		return res, fmt.Errorf("port %q has no canonical band (only api/ui/websocket switch)", portName)
	}
	cur := in.Ports[portName]
	res.PreviousPort = cur.Port
	if cur.Port > 0 {
		// Already fixed — leave it (never clobber a hand-pinned port).
		res.AssignedPort = cur.Port
		res.Changed = false
		res.Message = fmt.Sprintf("%s already has a fixed %s port (%d); no change.", res.Scenario, portName, cur.Port)
		return res, nil
	}

	used, err := occupiedFixedPorts(scenarioRoot)
	if err != nil {
		return res, err
	}
	start, end, err := parseBand(band)
	if err != nil {
		return res, err
	}
	chosen := 0
	for p := start; p <= end; p++ {
		if used[p] {
			continue
		}
		if listening != nil && listening(p) {
			continue
		}
		chosen = p
		break
	}
	if chosen == 0 {
		return res, fmt.Errorf("no free %s port available in band %s (all in use)", portName, band)
	}

	doc, before, err := loadServiceJSON(scenarioRoot)
	if err != nil {
		return res, err
	}
	ports := svcedit.EnsureMap(doc.Root(), "ports")
	port := svcedit.EnsureMap(ports, portName)
	port.Set("port", chosen)
	// A fixed port and a range are mutually exclusive in the canonical shape;
	// drop the range so the validator and the port resolver see one clean fixed
	// allocation.
	port.Delete("range")
	after, err := doc.Bytes()
	if err != nil {
		return res, err
	}
	res.AssignedPort = chosen
	res.Changed = true
	res.Before = string(before)
	res.After = string(after)
	res.Message = fmt.Sprintf("Assign %s fixed port %d to %s (band %s).", portName, chosen, res.Scenario, band)

	if apply {
		if err := writeServiceJSON(scenarioRoot, after); err != nil {
			return res, err
		}
		res.Applied = true
	}
	return res, nil
}

// ReleaseFixed reverts portName from a fixed allocation back to the canonical
// range, format-preserving. Idempotent: a port that is already ranged (no fixed
// port) is a no-op (Changed=false).
func ReleaseFixed(scenarioRoot, portName string, apply bool) (Result, error) {
	portName = normalizePortName(portName)
	res := Result{Scenario: scenarioName(scenarioRoot), PortName: portName}

	in, err := intent.Load(scenarioRoot)
	if err != nil {
		return res, fmt.Errorf("load service.json for %q: %w", res.Scenario, err)
	}
	band, _, ok := rules.CanonicalPortBand(portName, "")
	if !ok {
		return res, fmt.Errorf("port %q has no canonical band (only api/ui/websocket switch)", portName)
	}
	cur := in.Ports[portName]
	res.PreviousPort = cur.Port
	if cur.Port == 0 {
		res.Changed = false
		res.Message = fmt.Sprintf("%s %s port is already ranged; no change.", res.Scenario, portName)
		return res, nil
	}

	doc, before, err := loadServiceJSON(scenarioRoot)
	if err != nil {
		return res, err
	}
	ports := svcedit.EnsureMap(doc.Root(), "ports")
	port := svcedit.EnsureMap(ports, portName)
	port.Delete("port")
	// Restore the canonical range so the listener stays in-band when dynamically
	// allocated.
	if strings.TrimSpace(svcedit.StringField(port, "range")) == "" {
		port.Set("range", band)
	}
	after, err := doc.Bytes()
	if err != nil {
		return res, err
	}
	res.AssignedPort = 0
	res.Changed = true
	res.Before = string(before)
	res.After = string(after)
	res.Message = fmt.Sprintf("Release %s fixed port %d on %s back to range %s.", portName, cur.Port, res.Scenario, band)

	if apply {
		if err := writeServiceJSON(scenarioRoot, after); err != nil {
			return res, err
		}
		res.Applied = true
	}
	return res, nil
}

// occupiedFixedPorts collects every fixed listener port declared by the sibling
// scenarios of scenarioRoot (the scenario's own ports are excluded — assign only
// runs when the target is ranged anyway). It is the conflict map assign avoids.
func occupiedFixedPorts(scenarioRoot string) (map[int]bool, error) {
	used := map[int]bool{}
	scenariosRoot := filepath.Dir(scenarioRoot)
	self := filepath.Base(scenarioRoot)
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		// No sibling visibility (e.g. a standalone path) — return empty so assign
		// still works; the band reservation alone avoids cross-scenario clashes.
		return used, nil
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == self {
			continue
		}
		sib := filepath.Join(scenariosRoot, e.Name())
		if _, statErr := os.Stat(filepath.Join(sib, serviceJSONRel)); statErr != nil {
			continue
		}
		in, lerr := intent.Load(sib)
		if lerr != nil {
			continue
		}
		for _, p := range in.Ports {
			if p.Port > 0 {
				used[p.Port] = true
			}
		}
	}
	return used, nil
}

func loadServiceJSON(root string) (*svcedit.Doc, []byte, error) {
	path := filepath.Join(root, filepath.FromSlash(serviceJSONRel))
	raw, err := os.ReadFile(path) // #nosec G304 -- path built from a fixed root + .vrooli/service.json.
	if err != nil {
		return nil, nil, fmt.Errorf("read service.json: %w", err)
	}
	doc, err := svcedit.Parse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse service.json: %w", err)
	}
	return doc, raw, nil
}

func writeServiceJSON(root string, content []byte) error {
	path := filepath.Join(root, filepath.FromSlash(serviceJSONRel))
	if err := os.WriteFile(path, content, 0o644); err != nil { // #nosec G306 -- service.json is non-secret config.
		return fmt.Errorf("write service.json: %w", err)
	}
	return nil
}

func parseBand(band string) (int, int, error) {
	parts := strings.SplitN(strings.TrimSpace(band), "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("malformed band %q", band)
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || start <= 0 || end < start {
		return 0, 0, fmt.Errorf("malformed band %q", band)
	}
	return start, end, nil
}

func normalizePortName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return DefaultPortName
	}
	return name
}

func scenarioName(scenarioRoot string) string {
	return filepath.Base(strings.TrimRight(scenarioRoot, string(filepath.Separator)))
}

// DialListening is the production Listening probe: a short TCP dial to
// localhost:<port>. It lets assign avoid a port that something is already
// serving on, even if no scenario declares it fixed.
func DialListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
