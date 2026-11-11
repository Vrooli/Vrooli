// Package platformmatrix builds Image Tools support claims from observed platform
// inventory and checks recorded by the platform runner. It deliberately has no
// notion of a default-supported platform.
package platformmatrix

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const ReceiptSchema = "vrooli.image-tools.platform-receipt/v1"

type Target struct {
	ID          string `json:"id"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	Accelerator string `json:"accelerator,omitempty"`
}

type Node struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	OS           string   `json:"os"`
	Arch         string   `json:"arch"`
	Revision     string   `json:"revision"`
	Accelerators []string `json:"accelerators,omitempty"`
	Online       bool     `json:"online"`
	Dispatchable bool     `json:"dispatchable"`
}

type FleetSnapshot struct {
	Schema     string    `json:"schema"`
	ObservedAt time.Time `json:"observed_at"`
	Nodes      []Node    `json:"nodes"`
}

// Check is a raw execution record, not a support assertion. A zero exit code is
// accepted only when the runner also identifies a durable evidence artifact and
// its SHA-256 digest.
type Check struct {
	Command        string `json:"command"`
	ExitCode       int    `json:"exit_code"`
	EvidencePath   string `json:"evidence_path"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}

type Receipt struct {
	Schema       string           `json:"schema"`
	NodeID       string           `json:"node_id"`
	TargetID     string           `json:"target_id"`
	Revision     string           `json:"revision"`
	ObservedAt   time.Time        `json:"observed_at"`
	PackageCheck Check            `json:"package_check"`
	Capabilities map[string]Check `json:"capabilities"`
}

type Cell struct {
	CapabilityID string   `json:"capability_id"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason,omitempty"`
	NodeIDs      []string `json:"node_ids,omitempty"`
	Evidence     []string `json:"evidence,omitempty"`
}

type PlatformRow struct {
	TargetID string `json:"target_id"`
	GOOS     string `json:"goos"`
	GOARCH   string `json:"goarch"`
	Cells    []Cell `json:"capabilities"`
}

type Matrix struct {
	Schema           string        `json:"schema"`
	GeneratedAt      time.Time     `json:"generated_at"`
	CapabilitySource string        `json:"capability_source"`
	FleetObservedAt  time.Time     `json:"fleet_observed_at"`
	ReceiptCount     int           `json:"receipt_count"`
	Platforms        []PlatformRow `json:"platforms"`
}

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var capabilityRow = regexp.MustCompile(`^\|\s*(C[0-9]{2})\s*·\s*([^|]+)\|`)

func ParseCapabilities(markdown string) ([]string, error) {
	var ids []string
	seen := map[string]bool{}
	for _, line := range strings.Split(markdown, "\n") {
		m := capabilityRow.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		if seen[m[1]] {
			return nil, fmt.Errorf("duplicate capability ID %s", m[1])
		}
		seen[m[1]] = true
		ids = append(ids, m[1])
	}
	if len(ids) == 0 {
		return nil, errors.New("no Cnn capability rows found")
	}
	return ids, nil
}

func Build(targets []Target, fleet FleetSnapshot, receipts []Receipt, capabilities []string, source string, now time.Time) (Matrix, error) {
	if len(targets) == 0 || len(capabilities) == 0 {
		return Matrix{}, errors.New("targets and capabilities are required")
	}
	if fleet.Schema != "vrooli.fleet-platform-snapshot/v1" || fleet.ObservedAt.IsZero() {
		return Matrix{}, errors.New("fleet snapshot schema and observed_at are required")
	}
	targetByID := map[string]Target{}
	for _, target := range targets {
		if target.ID == "" || target.GOOS == "" || target.GOARCH == "" {
			return Matrix{}, errors.New("every target requires id, goos and goarch")
		}
		if _, exists := targetByID[target.ID]; exists {
			return Matrix{}, fmt.Errorf("duplicate target ID %s", target.ID)
		}
		targetByID[target.ID] = target
	}
	capabilitySet := map[string]bool{}
	for _, id := range capabilities {
		if id == "" || capabilitySet[id] {
			return Matrix{}, fmt.Errorf("empty or duplicate capability ID %q", id)
		}
		capabilitySet[id] = true
	}
	nodesByTarget := map[string][]Node{}
	nodeByID := map[string]Node{}
	for _, node := range fleet.Nodes {
		if node.ID == "" || node.OS == "" || node.Arch == "" {
			return Matrix{}, errors.New("fleet node requires id, os and arch")
		}
		if _, exists := nodeByID[node.ID]; exists {
			return Matrix{}, fmt.Errorf("duplicate fleet node ID %s", node.ID)
		}
		nodeByID[node.ID] = node
		for _, target := range targets {
			if node.OS == target.GOOS && node.Arch == target.GOARCH && hasAccelerator(node, target.Accelerator) {
				nodesByTarget[target.ID] = append(nodesByTarget[target.ID], node)
			}
		}
	}
	validReceipts := make(map[string]Receipt, len(receipts))
	for _, receipt := range receipts {
		if err := validateReceipt(receipt, targetByID, nodeByID, capabilitySet, fleet.ObservedAt); err != nil {
			return Matrix{}, err
		}
		key := receipt.TargetID + "\x00" + receipt.NodeID
		if _, exists := validReceipts[key]; exists {
			return Matrix{}, fmt.Errorf("duplicate receipt for target %s node %s", receipt.TargetID, receipt.NodeID)
		}
		validReceipts[key] = receipt
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	matrix := Matrix{Schema: "vrooli.image-tools.platform-matrix/v1", GeneratedAt: now.UTC(), CapabilitySource: source, FleetObservedAt: fleet.ObservedAt.UTC(), ReceiptCount: len(receipts)}
	for _, target := range targets {
		row := PlatformRow{TargetID: target.ID, GOOS: target.GOOS, GOARCH: target.GOARCH}
		for _, capabilityID := range capabilities {
			cell := Cell{CapabilityID: capabilityID}
			nodes := nodesByTarget[target.ID]
			if len(nodes) == 0 {
				cell.Status, cell.Reason = "unqualified", "no node"
				row.Cells = append(row.Cells, cell)
				continue
			}
			for _, node := range nodes {
				if !node.Online || !node.Dispatchable {
					continue
				}
				cell.NodeIDs = append(cell.NodeIDs, node.ID)
				receipt, ok := validReceipts[target.ID+"\x00"+node.ID]
				if !ok {
					continue
				}
				packageEvidence := receipt.PackageCheck.EvidencePath + "#sha256=" + receipt.PackageCheck.EvidenceSHA256
				if receipt.PackageCheck.ExitCode != 0 {
					cell.Evidence = append(cell.Evidence, packageEvidence)
					cell.Status = "failed"
					cell.Reason = "desktop package check failed"
					break
				}
				check, hasCheck := receipt.Capabilities[capabilityID]
				if !hasCheck {
					continue
				}
				cell.Evidence = append(cell.Evidence, check.EvidencePath+"#sha256="+check.EvidenceSHA256, packageEvidence)
				if check.ExitCode == 0 {
					cell.Status = "qualified"
					cell.Reason = "desktop package and capability check passed"
					break
				}
				cell.Status = "failed"
				cell.Reason = "desktop package or capability check failed"
			}
			if cell.Status == "" {
				cell.Status, cell.Reason = "unqualified", "no passing per-capability receipt"
			}
			sort.Strings(cell.NodeIDs)
			sort.Strings(cell.Evidence)
			row.Cells = append(row.Cells, cell)
		}
		matrix.Platforms = append(matrix.Platforms, row)
	}
	return matrix, nil
}

func hasAccelerator(node Node, want string) bool {
	if want == "" {
		return true
	}
	for _, accelerator := range node.Accelerators {
		if accelerator == want {
			return true
		}
	}
	return false
}

func validateReceipt(receipt Receipt, targets map[string]Target, nodes map[string]Node, capabilities map[string]bool, fleetObservedAt time.Time) error {
	if receipt.Schema != ReceiptSchema || receipt.NodeID == "" || receipt.Revision == "" || receipt.ObservedAt.IsZero() {
		return errors.New("receipt schema, node_id, revision and observed_at are required")
	}
	target, ok := targets[receipt.TargetID]
	if !ok {
		return fmt.Errorf("receipt references unknown target %q", receipt.TargetID)
	}
	node, ok := nodes[receipt.NodeID]
	if !ok || node.OS != target.GOOS || node.Arch != target.GOARCH {
		return fmt.Errorf("receipt node %q is absent from fleet snapshot or does not match target %q", receipt.NodeID, receipt.TargetID)
	}
	if receipt.Revision != node.Revision {
		return fmt.Errorf("receipt revision %q does not match node %s revision %q", receipt.Revision, node.ID, node.Revision)
	}
	if receipt.ObservedAt.After(fleetObservedAt) {
		return fmt.Errorf("receipt from %s is newer than fleet snapshot at %s", receipt.ObservedAt.UTC().Format(time.RFC3339Nano), fleetObservedAt.UTC().Format(time.RFC3339Nano))
	}
	if err := validateCheck(receipt.PackageCheck); err != nil {
		return fmt.Errorf("package check: %w", err)
	}
	if len(receipt.Capabilities) == 0 {
		return errors.New("receipt contains no capability checks")
	}
	for id, check := range receipt.Capabilities {
		if !capabilities[id] {
			return fmt.Errorf("receipt references unknown capability %q", id)
		}
		if err := validateCheck(check); err != nil {
			return fmt.Errorf("capability %s check: %w", id, err)
		}
	}
	return nil
}

func validateCheck(check Check) error {
	if strings.TrimSpace(check.Command) == "" || check.ExitCode < 0 || strings.TrimSpace(check.EvidencePath) == "" || !sha256Pattern.MatchString(check.EvidenceSHA256) {
		return errors.New("command, non-negative exit_code, evidence_path and lowercase SHA-256 are required")
	}
	return nil
}
