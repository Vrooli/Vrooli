package signals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// targetPassThreshold: an operational target passes when at least this
// fraction of its linked requirements pass (legacy TARGET_PASS_THRESHOLD).
const targetPassThreshold = 0.5

// requirementsCollector reads the requirements registry
// (requirements/index.json + imported module files) and the
// requirements-sync metadata under coverage/.
type requirementsCollector struct{ syncSource syncMetadataSource }

type syncMetadataSource interface {
	Load(context.Context, string, string) (*syncMetadata, error)
}

type emptySyncMetadataSource struct{}

func (emptySyncMetadataSource) Load(context.Context, string, string) (*syncMetadata, error) { return nil, nil }

type testGenieRequirementsSource struct{}

func (testGenieRequirementsSource) Load(ctx context.Context, scenario, _ string) (*syncMetadata, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		strings.TrimRight(baseURL, "/")+"/api/v1/scenarios/"+url.PathEscape(scenario)+"/requirements", nil)
	if err != nil {
		return nil, err
	}
	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("read test-genie requirements: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("read test-genie requirements: HTTP %d", response.StatusCode)
	}
	var snapshot struct {
		Modules []struct {
			Requirements []struct {
				ID         string `json:"id"`
				Status     string `json:"status"`
				LiveStatus string `json:"liveStatus"`
			} `json:"requirements"`
		} `json:"modules"`
	}
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("decode test-genie requirements: %w", err)
	}
	metadata := &syncMetadata{Requirements: make(map[string]syncedReq)}
	for _, module := range snapshot.Modules {
		for _, requirement := range module.Requirements {
			status := strings.TrimSpace(requirement.LiveStatus)
			if status == "" || strings.EqualFold(status, "unknown") {
				status = requirement.Status
			}
			if requirement.ID != "" && status != "" {
				metadata.Requirements[requirement.ID] = syncedReq{Status: status}
			}
		}
	}
	if len(metadata.Requirements) == 0 {
		return nil, nil
	}
	return metadata, nil
}

func (requirementsCollector) Name() string { return "requirements" }

func (c requirementsCollector) Collect(snap *Snapshot) error {
	dir := filepath.Join(snap.Root, "requirements")
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			// No registry is normal for unspecified scenarios: not
			// collected, not an error.
			return nil
		}
		return fmt.Errorf("stat requirements dir: %w", err)
	}

	reqs, err := loadRegistry(dir)
	if err != nil {
		return err
	}
	source := c.syncSource
	if source == nil {
		source = emptySyncMetadataSource{}
	}
	sync, err := source.Load(context.Background(), snap.Scenario, snap.Root)
	if err != nil {
		return err
	}
	snap.Requirements = summarizeRequirements(reqs, sync)
	return nil
}

// registryFile is the shared shape of index.json and module files: both may
// carry an imports list (paths relative to requirements/) and a
// requirements array.
type registryFile struct {
	Imports      []string          `json:"imports"`
	Requirements []requirementNode `json:"requirements"`
}

// requirementNode is one registry entry. children appears in two shapes in
// real registries: a list of requirement-ID strings referencing siblings in
// the same flat list, or inline child requirement objects (nested).
type requirementNode struct {
	ID                  string            `json:"id"`
	Status              string            `json:"status"`
	Priority            string            `json:"priority"`
	PRDRef              string            `json:"prd_ref"`
	OperationalTargetID string            `json:"operational_target_id"`
	Validation          []json.RawMessage `json:"validation"`
	DeliveryScope       string            `json:"delivery_scope"`
	Children            childList         `json:"children"`
}

// childList accepts both ID-reference children and inline child objects,
// including mixed arrays.
type childList struct {
	IDs   []string
	Nodes []requirementNode
}

func (c *childList) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("children: %w", err)
	}
	for _, item := range raw {
		trimmed := bytes.TrimSpace(item)
		if len(trimmed) == 0 {
			continue
		}
		switch trimmed[0] {
		case '"':
			var id string
			if err := json.Unmarshal(trimmed, &id); err != nil {
				return fmt.Errorf("children: %w", err)
			}
			c.IDs = append(c.IDs, id)
		case '{':
			var node requirementNode
			if err := json.Unmarshal(trimmed, &node); err != nil {
				return fmt.Errorf("children: %w", err)
			}
			c.Nodes = append(c.Nodes, node)
		default:
			return fmt.Errorf("children: unsupported element %s", trimmed)
		}
	}
	return nil
}

// flatReq is one flattened requirement: inline child objects are hoisted
// into the same list with parent links preserved via childIDs.
type flatReq struct {
	id              string
	status          string
	prdRef          string
	otID            string
	validationCount int
	deliveryScope   string
	childIDs        []string
}

// grouping nodes exist only to group children; they carry no status of
// their own and are excluded from pass counting.
func (r flatReq) grouping() bool {
	return len(r.childIDs) > 0 && r.status == ""
}

// loadRegistry loads requirements/index.json, its imports, and any
// module.json files found by recursive scan, deduplicated by path.
// Malformed JSON in any loaded file is an error (the degradation path);
// stale import references to missing files are skipped.
func loadRegistry(dir string) ([]flatReq, error) {
	loaded := map[string]bool{}
	var reqs []flatReq

	indexPath := filepath.Join(dir, "index.json")
	if data, err := os.ReadFile(indexPath); err == nil {
		loaded[indexPath] = true
		var idx registryFile
		if err := json.Unmarshal(data, &idx); err != nil {
			return nil, fmt.Errorf("decode %s: %w", indexPath, err)
		}
		reqs = append(reqs, flattenAll(idx.Requirements)...)
		for _, imp := range idx.Imports {
			more, err := loadRegistryFile(filepath.Join(dir, imp), loaded)
			if err != nil {
				return nil, err
			}
			reqs = append(reqs, more...)
		}
	}

	more, err := scanModules(dir, loaded)
	if err != nil {
		return nil, err
	}
	return append(reqs, more...), nil
}

func loadRegistryFile(path string, loaded map[string]bool) ([]flatReq, error) {
	if loaded[path] {
		return nil, nil
	}
	loaded[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		// Stale import reference; the rest of the registry still counts.
		return nil, nil
	}
	var f registryFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return flattenAll(f.Requirements), nil
}

func scanModules(dir string, loaded map[string]bool) ([]flatReq, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}

	var reqs []flatReq
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		switch {
		case entry.IsDir():
			more, err := scanModules(path, loaded)
			if err != nil {
				return nil, err
			}
			reqs = append(reqs, more...)
		case entry.Name() == "module.json":
			more, err := loadRegistryFile(path, loaded)
			if err != nil {
				return nil, err
			}
			reqs = append(reqs, more...)
		}
	}
	return reqs, nil
}

func flattenAll(nodes []requirementNode) []flatReq {
	var out []flatReq
	for i := range nodes {
		flatten(&nodes[i], &out)
	}
	return out
}

func flatten(n *requirementNode, out *[]flatReq) {
	fr := flatReq{
		id:              n.ID,
		status:          n.Status,
		prdRef:          n.PRDRef,
		otID:            n.OperationalTargetID,
		validationCount: len(n.Validation),
		deliveryScope:   normalizeDeliveryScope(n.DeliveryScope),
		childIDs:        append([]string(nil), n.Children.IDs...),
	}
	for j := range n.Children.Nodes {
		child := &n.Children.Nodes[j]
		if child.ID == "" {
			// Synthetic ID so anonymous inline children still contribute
			// to the depth tree.
			child.ID = fmt.Sprintf("%s/child-%d", n.ID, j)
		}
		fr.childIDs = append(fr.childIDs, child.ID)
	}
	*out = append(*out, fr)
	for j := range n.Children.Nodes {
		flatten(&n.Children.Nodes[j], out)
	}
}

// delivery scope is deliberately opt-in. Older registries have no field and
// must retain their committed-delivery behavior.
func normalizeDeliveryScope(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), "roadmap") {
		return "roadmap"
	}
	return "committed"
}

func (r flatReq) roadmap() bool { return r.deliveryScope == "roadmap" }

// syncMetadata is the requirements-sync artifact. Current writers
// (coverage/requirements-sync/latest.json) emit operational_targets with
// id/status/requirement_ids/completion_rate and no per-requirement map;
// legacy writers emitted a requirements status map and targets keyed by
// target_id/key with a counts object. Both shapes are accepted.
type syncMetadata struct {
	Requirements       map[string]syncedReq `json:"requirements"`
	OperationalTargets []syncTarget         `json:"operational_targets"`
}

type syncedReq struct {
	Status string `json:"status"`
}

type syncTarget struct {
	ID             string      `json:"id"`
	TargetID       string      `json:"target_id"`
	Key            string      `json:"key"`
	Status         string      `json:"status"`
	Counts         *syncCounts `json:"counts"`
	CompletionRate float64     `json:"completion_rate"`
	RequirementIDs []string    `json:"requirement_ids"`
}

type syncCounts struct {
	Total    int `json:"total"`
	Complete int `json:"complete"`
}

func summarizeRequirements(reqs []flatReq, sync *syncMetadata) RequirementsSignals {
	sig := RequirementsSignals{Collected: true}

	for _, r := range reqs {
		if r.grouping() || r.roadmap() {
			continue
		}
		sig.Total++
		if statusPasses(effectiveStatus(r, sync)) {
			sig.Passing++
		}
		if r.validationCount > 0 {
			sig.WithValidation++
		}
	}

	sig.TargetsTotal, sig.TargetsPassing = summarizeTargets(reqs, sync)
	sig.AvgDepth = averageDepth(reqs)
	return sig
}

// effectiveStatus: sync metadata wins over the registry file when it has an
// entry for the requirement.
func effectiveStatus(r flatReq, sync *syncMetadata) string {
	if sync != nil {
		if entry, ok := sync.Requirements[r.id]; ok && entry.Status != "" {
			return entry.Status
		}
	}
	return r.status
}

// statusPasses mirrors test-genie's NormalizeDeclaredStatus complete set
// (complete/completed/done/implemented) plus the passing-evidence forms the
// fleet's registries use (passed/passing/validated). Keep in lockstep with
// test-genie internal/requirements/types/status.go.
func statusPasses(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "passing", "complete", "completed", "done", "implemented", "validated":
		return true
	default:
		return false
	}
}

// summarizeTargets prefers sync operational_targets; otherwise it groups
// requirements by operational_target_id / prd_ref and applies the legacy
// >=50% member-pass threshold.
func summarizeTargets(reqs []flatReq, sync *syncMetadata) (total, passing int) {
	if sync != nil && len(sync.OperationalTargets) > 0 {
		for _, t := range sync.OperationalTargets {
			if !targetHasCommittedRequirement(t, reqs) {
				continue
			}
			total++
			if targetPassesSync(t) {
				passing++
			}
		}
		return total, passing
	}

	type tally struct{ pass, total int }
	groups := map[string]*tally{}
	for _, r := range reqs {
		if r.grouping() || r.roadmap() {
			continue
		}
		key := r.otID
		if key == "" {
			key = targetFromPRDRef(r.prdRef)
		}
		if key == "" {
			continue
		}
		g, ok := groups[key]
		if !ok {
			g = &tally{}
			groups[key] = g
		}
		g.total++
		if statusPasses(effectiveStatus(r, sync)) {
			g.pass++
		}
	}

	for _, g := range groups {
		total++
		if g.total > 0 && float64(g.pass)/float64(g.total) >= targetPassThreshold {
			passing++
		}
	}
	return total, passing
}

func targetHasCommittedRequirement(target syncTarget, reqs []flatReq) bool {
	byID := make(map[string]flatReq, len(reqs))
	for _, req := range reqs {
		byID[req.id] = req
	}
	foundRequirement := false
	for _, id := range target.RequirementIDs {
		req, ok := byID[id]
		if !ok {
			continue
		}
		foundRequirement = true
		if !req.grouping() && !req.roadmap() {
			return true
		}
	}
	// Older sync artifacts did not consistently carry requirement IDs. Keep
	// their historical scoring behavior; only exclude a target when its known
	// linked requirements are explicitly all roadmap work.
	return !foundRequirement
}

func targetPassesSync(t syncTarget) bool {
	if statusPasses(t.Status) {
		return true
	}
	if t.Counts != nil && t.Counts.Total > 0 {
		return float64(t.Counts.Complete)/float64(t.Counts.Total) > targetPassThreshold
	}
	// Current writers emit completion_rate as a 0-100 percentage.
	return t.CompletionRate > targetPassThreshold*100
}

// targetFromPRDRef extracts OT-P0-001 style target IDs from prd_ref.
func targetFromPRDRef(prdRef string) string {
	if prdRef == "" {
		return ""
	}
	upper := strings.ToUpper(prdRef)
	if strings.HasPrefix(upper, "OT-P") {
		return upper
	}
	return ""
}

// averageDepth is the mean max-depth over root requirement trees (nodes
// never referenced as a child). Flat lists yield 1.0.
func averageDepth(reqs []flatReq) float64 {
	if len(reqs) == 0 {
		return 0
	}

	childMap := map[string][]string{}
	hasParent := map[string]bool{}
	for _, r := range reqs {
		if len(r.childIDs) > 0 {
			childMap[r.id] = r.childIDs
		}
		for _, c := range r.childIDs {
			hasParent[c] = true
		}
	}

	totalDepth, roots := 0, 0
	for _, r := range reqs {
		if hasParent[r.id] {
			continue
		}
		roots++
		totalDepth += maxDepth(r.id, childMap, map[string]bool{})
	}
	if roots == 0 {
		return 0
	}
	return float64(totalDepth) / float64(roots)
}

// maxDepth guards against reference cycles in malformed registries.
func maxDepth(id string, childMap map[string][]string, onPath map[string]bool) int {
	if onPath[id] {
		return 0
	}
	onPath[id] = true
	defer delete(onPath, id)

	deepest := 0
	for _, child := range childMap[id] {
		if d := maxDepth(child, childMap, onPath); d > deepest {
			deepest = d
		}
	}
	return 1 + deepest
}
