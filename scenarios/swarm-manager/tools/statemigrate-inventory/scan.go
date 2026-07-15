package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Config selects the roots to scan. Empty fields fall back to the resolved live
// roots (see resolveRoots).
type Config struct {
	DataRoot   string
	StateRoot  string
	CacheRoot  string
	ConfigFile string
	// ResolvedFrom is a human note recording how the roots were chosen (env vs
	// default). It is echoed into Roots.ResolvedFrom.
	ResolvedFrom string
}

const schemaVersion = "swarm-statemigrate-inventory/v1"

// backlogKindDir maps the on-disk kind directory to the canonical kind label.
var backlogKindDir = map[string]string{
	"ideas":    "idea",
	"research": "research",
	"fix":      "fix",
	"execute":  "execute",
	"chore":    "chore",
}

// validBacklogStatus mirrors internal/backlogstatus.All() exactly (the canonical
// enum enforced by backlog.LoadItemFromPath). A spec.json carrying anything else
// is reported as an anomaly (never normalized away). Archival is a separate
// archived_at flag, not a status value.
var validBacklogStatus = map[string]bool{
	"suggested": true, "backlog": true, "researching": true, "ready": true,
	"queued": true, "in_progress": true, "in_review": true, "review_pending": true,
	"completed": true, "failed": true, "needs_followup": true,
}

// validInitiativeStatus mirrors internal/initiatives model.go.
var validInitiativeStatus = map[string]bool{
	"active": true, "in_review": true, "review_pending": true,
	"completed": true, "failed": true, "needs_followup": true,
}

// validGoalStatus mirrors internal/goals model.go.
var validGoalStatus = map[string]bool{"active": true, "archived": true}

// classKind maps a class name to its aggregation kind.
var classKind = map[string]string{
	"backlog_item":              "primary",
	"initiative":                "primary",
	"goal":                      "primary",
	"record":                    "primary",
	"om_execution_manifest":     "primary",
	"capture":                   "primary",
	"execution_runs":            "state",
	"engagement_owners":         "state",
	"circuit_breaker":           "state",
	"agent_activities":          "state",
	"queue":                     "state",
	"om_global_run_owners":      "state",
	"om_scope_run_owners":       "state",
	"eventlog_sqlite":           "opaque",
	"foreign_deployment_report": "foreign",
}

// planRef is the shared plan-manager reference shape.
type planRef struct {
	Provider string `json:"provider"`
	PlanID   string `json:"plan_id"`
	Slug     string `json:"slug"`
	Role     string `json:"role"`
}

// scanner accumulates state during a scan.
type scanner struct {
	inv         *Inventory
	classFiles  map[string][]string // class -> sorted "relpath\x00filehash" lines
	classBytes  map[string]int64
	classCount  map[string]int
	primary     map[string][]ObjectRecord // class -> objects
	byStatus    map[string]map[string]int
	byKind      map[string]map[string]int
	withPlanRef map[string]int
	allLines    []string // global "root/relpath\x00filehash"

	// graph state for referential checks
	itemNames       map[string]bool   // "kind/name"
	itemInitiative  map[string]string // "kind/name" -> initiative
	initiativeNames map[string]bool
	initiativeItems map[string][]string // initiative -> ["kind/name"]

	planRefs OwnedPlanRefs
}

// OwnedPlanRefs is a list of (owner, ref) pairs gathered during the scan.
type OwnedPlanRefs []struct {
	Owner string
	Ref   planRef
}

// Scan walks the configured roots and returns a deterministic Inventory. It never
// aborts on unreadable state: read/parse failures become Anomaly records.
func Scan(cfg Config) *Inventory {
	s := &scanner{
		inv:             &Inventory{SchemaVersion: schemaVersion},
		classFiles:      map[string][]string{},
		classBytes:      map[string]int64{},
		classCount:      map[string]int{},
		primary:         map[string][]ObjectRecord{},
		byStatus:        map[string]map[string]int{},
		byKind:          map[string]map[string]int{},
		withPlanRef:     map[string]int{},
		itemNames:       map[string]bool{},
		itemInitiative:  map[string]string{},
		initiativeNames: map[string]bool{},
		initiativeItems: map[string][]string{},
	}

	s.inv.Roots = Roots{
		ResolvedFrom: cfg.ResolvedFrom,
		Data:         rootStatus(cfg.DataRoot),
		State:        rootStatus(cfg.StateRoot),
		Cache:        rootStatus(cfg.CacheRoot),
		ConfigFile:   rootStatus(cfg.ConfigFile),
	}
	s.inv.Roots.ShadowNamespacesPresent = detectShadowNamespaces(cfg)

	s.walkRoot("data", cfg.DataRoot)
	s.walkRoot("state", cfg.StateRoot)
	s.walkRoot("cache", cfg.CacheRoot)
	s.walkConfigFile(cfg.ConfigFile)

	s.checkExpectedAbsent(cfg)
	s.computeReferential()
	s.computePlanRefs()
	s.finalize()
	return s.inv
}

func rootStatus(p string) RootStatus {
	if p == "" {
		return RootStatus{Path: "", Exists: false}
	}
	_, err := os.Stat(p)
	return RootStatus{Path: redactHome(p), Exists: err == nil}
}

func detectShadowNamespaces(cfg Config) []string {
	var out []string
	for _, p := range []string{cfg.DataRoot, cfg.StateRoot, cfg.CacheRoot} {
		if p == "" {
			continue
		}
		shadow := p + "_shadow"
		if _, err := os.Stat(shadow); err == nil {
			out = append(out, redactHome(shadow))
		}
	}
	sort.Strings(out)
	return uniqueSorted(out)
}

// walkRoot walks one root, hashing every file and attributing it to a class.
func (s *scanner) walkRoot(rootName, rootPath string) {
	if rootPath == "" {
		return
	}
	if _, err := os.Stat(rootPath); err != nil {
		return // absence handled by checkExpectedAbsent / Roots
	}
	_ = filepath.WalkDir(rootPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			s.addAnomaly("walk_error", rootName+"/"+relOf(rootPath, p), err.Error())
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel := relOf(rootPath, p)
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			s.addAnomaly("read_error", rootName+"/"+rel, readErr.Error())
			return nil
		}
		sum := sha256.Sum256(data)
		hexsum := hex.EncodeToString(sum[:])
		size := int64(len(data))

		class, kind, primary := classify(rootName, rel)
		s.classCount[class]++
		s.classBytes[class] += size
		s.classFiles[class] = append(s.classFiles[class], rel+"\x00"+hexsum)
		s.allLines = append(s.allLines, rootName+"/"+rel+"\x00"+hexsum)

		if class == "unclassified" {
			s.addFinding("unclassified_artifact", rootName+"/"+rel, "", "file matched no known swarm-manager storage pattern; investigate before migration")
		}
		if primary {
			s.parsePrimary(class, kind, rootName, rel, data, hexsum, size)
		} else {
			s.parseIndex(class, rootName, rel, data)
		}
		return nil
	})
}

func (s *scanner) walkConfigFile(p string) {
	if p == "" {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return // absence recorded in Roots.ConfigFile
	}
	sum := sha256.Sum256(data)
	hexsum := hex.EncodeToString(sum[:])
	s.classCount["settings_config"]++
	s.classBytes["settings_config"] += int64(len(data))
	s.classFiles["settings_config"] = append(s.classFiles["settings_config"], "settings.json\x00"+hexsum)
	s.allLines = append(s.allLines, "config/settings.json\x00"+hexsum)
	var probe map[string]any
	if json.Unmarshal(data, &probe) != nil {
		s.addAnomaly("corrupt_json", "config/settings.json", "settings config is not valid JSON")
	}
}

// classify returns (class, kindLabel, isPrimaryDocument) for a file.
func classify(root, rel string) (string, string, bool) {
	base := path.Base(rel)
	seg := strings.Split(rel, "/")
	switch root {
	case "data":
		if kindLabel, ok := backlogKindDir[seg[0]]; ok {
			if len(seg) == 3 && base == "spec.json" {
				return "backlog_item", kindLabel, true
			}
			switch {
			case containsSeq(seg, "workshop", "clarifications"):
				return "workshop_clarification", "", false
			case contains(seg, "workshop") && isRoundFile(base):
				return "workshop_round", "", false
			case base == "acceptance-validation.json":
				return "acceptance_validation", "", false
			case contains(seg, "review"):
				return "backlog_review_artifact", "", false
			case contains(seg, "evidence"):
				return "backlog_evidence_artifact", "", false
			case contains(seg, ".swarm"):
				return "item_swarm_artifact", "", false
			case contains(seg, "clarify"):
				return "backlog_clarify_artifact", "", false
			default:
				return "item_doc", "", false
			}
		}
		switch seg[0] {
		case "initiatives":
			switch {
			case len(seg) == 3 && base == "initiative.json":
				return "initiative", "", true
			case base == "graph.json":
				return "initiative_graph", "", false
			case containsSeq(seg, "review", "decisions"):
				return "initiative_decision", "", false
			case contains(seg, "review") && isRoundFile(base):
				return "initiative_review_round", "", false
			case contains(seg, "modes"):
				return "operating_mode_initiative", "", false
			case base == ".feedback-lock":
				return "initiative_lock", "", false
			default:
				return "initiative_context_file", "", false
			}
		case "goals":
			if len(seg) == 3 && base == "goal.json" {
				return "goal", "", true
			}
			return "goal_artifact", "", false
		case "records":
			if strings.HasSuffix(base, ".json") {
				return "record", "", true
			}
			return "record_artifact", "", false
		case "mode-targets":
			switch {
			case base == "manifest.json":
				return "om_execution_manifest", "", true
			case base == "run-owners.json":
				return "om_scope_run_owners", "", false
			case contains(seg, "legacy-rounds"):
				return "om_legacy_round", "", false
			case isRoundFile(base):
				return "om_round", "", false
			default:
				return "mode_target_artifact", "", false
			}
		case "operating-mode-run-owners":
			return "om_global_run_owners", "", false
		case "deployment":
			return "foreign_deployment_report", "", false
		case "autofiler":
			return "autofiler_state", "", false
		}
		switch {
		case base == "auto-drain.json":
			return "autodrain_state", "", false
		case base == "plan-ref-sweep-manifest.jsonl":
			return "plan_ref_sweep_manifest", "", false
		case strings.HasPrefix(base, "events.db"):
			return "eventlog_sqlite", "", false
		}
		return "unclassified", "", false
	case "state":
		switch base {
		case "execution-runs.json":
			return "execution_runs", "", false
		case "engagement-owners.json":
			return "engagement_owners", "", false
		case "circuit-breaker.json":
			return "circuit_breaker", "", false
		case "agent-activities.json":
			return "agent_activities", "", false
		case "queue.json":
			return "queue", "", false
		}
		return "unclassified", "", false
	case "cache":
		if seg[0] == "captures" {
			switch {
			case base == "capture.json":
				return "capture", "", true
			case base == "classification.json":
				return "capture_classification", "", false
			case contains(seg, "attachments"):
				return "capture_attachment", "", false
			default:
				return "capture_artifact", "", false
			}
		}
		return "unclassified", "", false
	}
	return "unclassified", "", false
}

// parsePrimary decodes a primary document, records its identity/status/refs, and
// reports invalid values as anomalies.
func (s *scanner) parsePrimary(class, kindLabel, rootName, rel string, data []byte, hexsum string, size int64) {
	rec := ObjectRecord{RelPath: rel, Size: size, SHA256: hexsum}
	switch class {
	case "backlog_item":
		var it struct {
			Name       string   `json:"name"`
			Status     string   `json:"status"`
			Initiative string   `json:"initiative"`
			DependsOn  []string `json:"depends_on"`
			PlanRef    *planRef `json:"plan_ref"`
		}
		if err := json.Unmarshal(data, &it); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "backlog spec.json unparseable: "+err.Error())
			s.appendPrimary(class, ObjectRecord{Identity: kindLabel + "/" + dirName(rel), RelPath: rel, Size: size, SHA256: hexsum, Status: "<unparseable>"})
			return
		}
		name := it.Name
		if name == "" {
			name = dirName(rel)
		}
		id := kindLabel + "/" + name
		rec.Identity = id
		rec.Status = it.Status
		s.itemNames[id] = true
		if strings.TrimSpace(it.Initiative) != "" {
			s.itemInitiative[id] = it.Initiative
			rec.Refs = append(rec.Refs, "initiative:"+it.Initiative)
		}
		for _, d := range it.DependsOn {
			rec.Refs = append(rec.Refs, "depends_on:"+d)
		}
		if !validBacklogStatus[it.Status] {
			s.addAnomaly("invalid_status", rootName+"/"+rel, "backlog status "+jsonQuote(it.Status)+" not in canonical enum")
		}
		s.bumpStatus(class, it.Status)
		s.bumpKind(class, kindLabel)
		if it.PlanRef != nil {
			s.withPlanRef[class]++
			s.planRefs = append(s.planRefs, struct {
				Owner string
				Ref   planRef
			}{id, *it.PlanRef})
			rec.Refs = append(rec.Refs, "plan_ref:"+it.PlanRef.Provider+"/"+it.PlanRef.PlanID)
		}
	case "initiative":
		var in struct {
			Name      string   `json:"name"`
			Status    string   `json:"status"`
			Mode      string   `json:"mode"`
			Items     []string `json:"items"`
			DependsOn []string `json:"depends_on"`
			PlanRef   *planRef `json:"plan_ref"`
		}
		if err := json.Unmarshal(data, &in); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "initiative.json unparseable: "+err.Error())
			s.appendPrimary(class, ObjectRecord{Identity: dirName(rel), RelPath: rel, Size: size, SHA256: hexsum, Status: "<unparseable>"})
			return
		}
		name := in.Name
		if name == "" {
			name = dirName(rel)
		}
		rec.Identity = name
		rec.Status = in.Status
		s.initiativeNames[name] = true
		s.initiativeItems[name] = append(s.initiativeItems[name], in.Items...)
		for _, m := range in.Items {
			rec.Refs = append(rec.Refs, "item:"+m)
		}
		for _, d := range in.DependsOn {
			rec.Refs = append(rec.Refs, "depends_on:"+d)
		}
		if !validInitiativeStatus[in.Status] {
			s.addAnomaly("invalid_status", rootName+"/"+rel, "initiative status "+jsonQuote(in.Status)+" not in canonical enum")
		}
		s.bumpStatus(class, in.Status)
		if in.PlanRef != nil {
			s.withPlanRef[class]++
			s.planRefs = append(s.planRefs, struct {
				Owner string
				Ref   planRef
			}{"initiative/" + name, *in.PlanRef})
			rec.Refs = append(rec.Refs, "plan_ref:"+in.PlanRef.Provider+"/"+in.PlanRef.PlanID)
		}
	case "goal":
		var g struct {
			Name    string   `json:"name"`
			Status  string   `json:"status"`
			Targets []string `json:"targets"`
		}
		if err := json.Unmarshal(data, &g); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "goal.json unparseable: "+err.Error())
			s.appendPrimary(class, ObjectRecord{Identity: dirName(rel), RelPath: rel, Size: size, SHA256: hexsum, Status: "<unparseable>"})
			return
		}
		name := g.Name
		if name == "" {
			name = dirName(rel)
		}
		rec.Identity = name
		rec.Status = g.Status
		for _, t := range g.Targets {
			rec.Refs = append(rec.Refs, "target:"+t)
		}
		if !validGoalStatus[g.Status] {
			s.addAnomaly("invalid_status", rootName+"/"+rel, "goal status "+jsonQuote(g.Status)+" not in {active,archived}")
		}
		s.bumpStatus(class, g.Status)
	case "record":
		var r struct {
			ID           string `json:"id"`
			Kind         string `json:"kind"`
			BacklogRef   string `json:"backlog_ref"`
			InitiativeID string `json:"initiative_id"`
			Outcome      string `json:"outcome"`
		}
		if err := json.Unmarshal(data, &r); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "record json unparseable: "+err.Error())
			s.appendPrimary(class, ObjectRecord{Identity: strings.TrimSuffix(base(rel), ".json"), RelPath: rel, Size: size, SHA256: hexsum, Status: "<unparseable>"})
			return
		}
		id := r.ID
		if id == "" {
			id = strings.TrimSuffix(base(rel), ".json")
		}
		rec.Identity = id
		rec.Status = r.Outcome
		if strings.TrimSpace(r.BacklogRef) != "" {
			rec.Refs = append(rec.Refs, "backlog_ref:"+r.BacklogRef)
		}
		if strings.TrimSpace(r.InitiativeID) != "" {
			rec.Refs = append(rec.Refs, "initiative_id:"+r.InitiativeID)
		}
		s.bumpStatus(class, r.Outcome)
		if r.Kind != "" {
			s.bumpKind(class, r.Kind)
		}
	case "om_execution_manifest":
		var m struct {
			ExecutionID string `json:"execution_id"`
			ScopeKind   string `json:"scope_kind"`
			ScopeID     string `json:"scope_id"`
			Mode        string `json:"mode"`
			Status      string `json:"status"`
		}
		if err := json.Unmarshal(data, &m); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "operating-mode manifest unparseable: "+err.Error())
			s.appendPrimary(class, ObjectRecord{Identity: rel, RelPath: rel, Size: size, SHA256: hexsum, Status: "<unparseable>"})
			return
		}
		rec.Identity = m.ExecutionID
		if rec.Identity == "" {
			rec.Identity = rel
		}
		rec.Status = m.Status
		rec.Refs = append(rec.Refs, "scope:"+m.ScopeKind+"/"+m.ScopeID, "mode:"+m.Mode)
		s.bumpStatus(class, m.Status)
	case "capture":
		var c struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &c); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "capture.json unparseable: "+err.Error())
			s.appendPrimary(class, ObjectRecord{Identity: dirName(rel), RelPath: rel, Size: size, SHA256: hexsum, Status: "<unparseable>"})
			return
		}
		rec.Identity = c.ID
		if rec.Identity == "" {
			rec.Identity = dirName(rel)
		}
		rec.Status = c.Status
		s.bumpStatus(class, c.Status)
	default:
		rec.Identity = rel
	}
	s.appendPrimary(class, rec)
}

// parseIndex decodes the run-owner / engagement / execution index files that
// feed ownership-ambiguity checks. Corrupt index files are anomalies.
func (s *scanner) parseIndex(class, rootName, rel string, data []byte) {
	switch class {
	case "om_global_run_owners":
		s.inv.Ownership.GlobalRunOwnerIndexPresent = true
		var idx struct {
			Owners map[string][]struct {
				TargetKind  string `json:"target_kind"`
				ScopeID     string `json:"scope_id"`
				Mode        string `json:"mode"`
				ExecutionID string `json:"execution_id"`
				Round       int    `json:"round"`
			} `json:"owners"`
		}
		if err := json.Unmarshal(data, &idx); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "global run-owner index unparseable: "+err.Error())
			return
		}
		for runID, owners := range idx.Owners {
			if len(owners) > 1 {
				var names []string
				for _, o := range owners {
					names = append(names, o.TargetKind+"/"+o.ScopeID+"#"+o.ExecutionID)
				}
				sort.Strings(names)
				s.inv.Ownership.AmbiguousRunOwners = append(s.inv.Ownership.AmbiguousRunOwners, AmbiguousOwner{
					RunID: runID, Owners: uniqueSorted(names), Source: "operating-mode-run-owners/run-owners.json",
				})
			}
		}
	case "om_scope_run_owners":
		s.inv.Ownership.ScopeRunOwnerIndexes++
		var probe map[string]any
		if json.Unmarshal(data, &probe) != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "scope run-owner index unparseable")
		}
	case "engagement_owners":
		s.inv.Ownership.EngagementOwnersPresent = true
		var m map[string]struct {
			Engagements map[string]string `json:"engagements"`
		}
		if err := json.Unmarshal(data, &m); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "engagement-owners unparseable: "+err.Error())
			return
		}
		// A scenario engaged under more than one owner is an exclusivity violation.
		scenarioOwners := map[string][]string{}
		for owner, set := range m {
			for scenario := range set.Engagements {
				scenarioOwners[scenario] = append(scenarioOwners[scenario], owner)
			}
		}
		for scenario, owners := range scenarioOwners {
			if len(owners) > 1 {
				s.inv.Ownership.AmbiguousRunOwners = append(s.inv.Ownership.AmbiguousRunOwners, AmbiguousOwner{
					RunID: "engagement:" + scenario, Owners: uniqueSorted(owners), Source: "engagement-owners.json",
				})
			}
		}
	case "execution_runs":
		var recs []struct {
			ExecutionID string `json:"execution_id"`
			BacklogKind string `json:"backlog_kind"`
			BacklogName string `json:"backlog_name"`
			RunID       string `json:"run_id"`
			Status      string `json:"status"`
		}
		if err := json.Unmarshal(data, &recs); err != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, "execution-runs.json unparseable: "+err.Error())
			return
		}
		bs := map[string]int{}
		for _, r := range recs {
			bs[r.Status]++
			if r.Status == "running" && strings.TrimSpace(r.RunID) == "" {
				s.addFinding("orphaned_execution", "execution/"+r.ExecutionID, r.BacklogKind+"/"+r.BacklogName, "status=running with empty agent-manager run_id")
			}
		}
		if len(bs) > 0 {
			s.byStatus["execution_runs"] = bs
		}
	case "circuit_breaker", "agent_activities", "queue":
		var probe any
		if json.Unmarshal(data, &probe) != nil {
			s.addAnomaly("corrupt_json", rootName+"/"+rel, class+" state file is not valid JSON")
		}
	}
}

// computeReferential derives dangling-reference and divergence findings from the
// parsed object graph.
func (s *scanner) computeReferential() {
	// backlog item -> initiative existence and reverse membership.
	for id, initName := range s.itemInitiative {
		if !s.initiativeNames[initName] {
			s.addFinding("dangling_item_initiative", id, "initiative/"+initName, "item.initiative points to a non-existent initiative")
		}
	}
	// backlog item depends_on existence.
	for _, rec := range s.primary["backlog_item"] {
		for _, ref := range rec.Refs {
			dep, ok := strings.CutPrefix(ref, "depends_on:")
			if !ok {
				continue
			}
			if !s.itemNames[dep] {
				s.addFinding("dangling_dependency", rec.Identity, dep, "depends_on target not found on disk")
			}
		}
	}
	// initiative membership existence + reverse divergence.
	for initName, items := range s.initiativeItems {
		for _, member := range items {
			if !s.itemNames[member] {
				s.addFinding("dangling_initiative_item", "initiative/"+initName, member, "initiative.items references a non-existent backlog item")
				continue
			}
			if s.itemInitiative[member] != initName {
				s.addFinding("initiative_membership_divergence", "initiative/"+initName, member,
					"initiative lists item but item.initiative="+jsonQuote(s.itemInitiative[member]))
			}
		}
	}
	// initiative depends_on existence.
	for _, rec := range s.primary["initiative"] {
		for _, ref := range rec.Refs {
			dep, ok := strings.CutPrefix(ref, "depends_on:")
			if !ok {
				continue
			}
			if !s.initiativeNames[dep] {
				s.addFinding("dangling_initiative_dependency", "initiative/"+rec.Identity, "initiative/"+dep, "initiative.depends_on target not found")
			}
		}
	}
	// goal targets existence.
	for _, rec := range s.primary["goal"] {
		for _, ref := range rec.Refs {
			tgt, ok := strings.CutPrefix(ref, "target:")
			if !ok {
				continue
			}
			if strings.HasPrefix(tgt, "initiative/") {
				name := strings.TrimPrefix(tgt, "initiative/")
				if !s.initiativeNames[name] {
					s.addFinding("dangling_goal_target", "goal/"+rec.Identity, tgt, "goal target initiative not found")
				}
			} else if !s.itemNames[tgt] {
				s.addFinding("dangling_goal_target", "goal/"+rec.Identity, tgt, "goal target backlog item not found")
			}
		}
	}
	// foreign deployment report -> ambiguous ownership finding.
	if s.classCount["foreign_deployment_report"] > 0 {
		s.addFinding("ambiguous_ownership", "data/deployment/deployment-report.json", "",
			"foreign artifact written by scenario-dependency-analyzer under swarm-manager data root; not swarm-manager-owned state")
	}
}

// computePlanRefs classifies gathered plan-refs into managed vs unmanaged.
func (s *scanner) computePlanRefs() {
	sort.Slice(s.planRefs, func(i, j int) bool { return s.planRefs[i].Owner < s.planRefs[j].Owner })
	sum := PlanRefSummary{Details: []PlanRefDetail{}}
	knownRole := map[string]bool{"execution_spec": true, "operating_mode_plan": true}
	for _, pr := range s.planRefs {
		sum.Total++
		r := pr.Ref
		var reasons []string
		if r.Provider != "plan-manager" {
			reasons = append(reasons, "provider="+jsonQuote(r.Provider))
		}
		if strings.TrimSpace(r.PlanID) == "" {
			reasons = append(reasons, "empty plan_id")
		}
		if r.Role != "" && !knownRole[r.Role] {
			reasons = append(reasons, "unknown role="+jsonQuote(r.Role))
		}
		if len(reasons) == 0 {
			sum.Managed++
			continue
		}
		sum.Unmanaged++
		sum.Details = append(sum.Details, PlanRefDetail{
			Owner: pr.Owner, Provider: r.Provider, PlanID: r.PlanID, Role: r.Role,
			Reason: strings.Join(reasons, "; "),
		})
	}
	sort.Slice(sum.Details, func(i, j int) bool { return sum.Details[i].Owner < sum.Details[j].Owner })
	s.inv.PlanRefs = sum
}

// checkExpectedAbsent records known state files the code writes that are missing
// on disk, so their absence is explicit rather than an unscanned gap.
func (s *scanner) checkExpectedAbsent(cfg Config) {
	expect := []struct{ root, rel, note string }{
		{"state", "execution-runs.json", "item-level execution run log; absent means no runs recorded (or recently pruned)"},
		{"state", "engagement-owners.json", "engagement/run-owner exclusivity index; absent means no open engagements"},
		{"state", "circuit-breaker.json", "per-item failure circuit breaker; absent means no trips recorded"},
		{"state", "queue.json", "execution queue snapshot; absent means empty queue"},
		{"state", "agent-activities.json", "agent activity ledger"},
		{"data", "operating-mode-run-owners/run-owners.json", "global operating-mode run-owner index; absent means no mode-round runs indexed"},
		{"data", "auto-drain.json", "auto-drain flag; absent means disabled (default)"},
		{"data", "autofiler/dismissed_findings.json", "auto-filer dismissals; absent means none dismissed"},
	}
	rootPath := map[string]string{"state": cfg.StateRoot, "data": cfg.DataRoot}
	for _, e := range expect {
		rp := rootPath[e.root]
		if rp == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(rp, filepath.FromSlash(e.rel))); err != nil {
			s.inv.ExpectedAbsent = append(s.inv.ExpectedAbsent, ExpectedAbsent{RelPath: e.rel, Root: e.root, Note: e.note})
		}
	}
	sort.Slice(s.inv.ExpectedAbsent, func(i, j int) bool {
		if s.inv.ExpectedAbsent[i].Root != s.inv.ExpectedAbsent[j].Root {
			return s.inv.ExpectedAbsent[i].Root < s.inv.ExpectedAbsent[j].Root
		}
		return s.inv.ExpectedAbsent[i].RelPath < s.inv.ExpectedAbsent[j].RelPath
	})
}

// finalize assembles sorted class inventories, hashes, and totals.
func (s *scanner) finalize() {
	var totalBytes int64
	var totalFiles int
	classes := make([]string, 0, len(s.classCount))
	for c := range s.classCount {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	for _, c := range classes {
		lines := s.classFiles[c]
		sort.Strings(lines)
		ci := ClassInventory{
			Class:       c,
			Kind:        classKindOf(c),
			Count:       s.classCount[c],
			Bytes:       s.classBytes[c],
			ContentHash: "sha256:" + hashLines(lines),
		}
		if bs := s.byStatus[c]; len(bs) > 0 {
			ci.ByStatus = bs
		}
		if bk := s.byKind[c]; len(bk) > 0 {
			ci.ByKind = bk
		}
		if wp := s.withPlanRef[c]; wp > 0 {
			ci.WithPlanRef = wp
		}
		if objs := s.primary[c]; len(objs) > 0 {
			sort.Slice(objs, func(i, j int) bool { return objs[i].Identity < objs[j].Identity })
			for i := range objs {
				sort.Strings(objs[i].Refs)
			}
			ci.Objects = objs
			s.inv.Totals.ObjectCount += len(objs)
		}
		s.inv.Classes = append(s.inv.Classes, ci)
		totalBytes += s.classBytes[c]
		totalFiles += s.classCount[c]
	}

	sortFindings(s.inv.ReferentialFindings)
	sortAnomalies(s.inv.Anomalies)
	sort.Slice(s.inv.Ownership.AmbiguousRunOwners, func(i, j int) bool {
		return s.inv.Ownership.AmbiguousRunOwners[i].RunID < s.inv.Ownership.AmbiguousRunOwners[j].RunID
	})
	if s.inv.Ownership.AmbiguousRunOwners == nil {
		s.inv.Ownership.AmbiguousRunOwners = []AmbiguousOwner{}
	}

	sort.Strings(s.allLines)
	s.inv.Totals.FilesScanned = totalFiles
	s.inv.Totals.Bytes = totalBytes
	s.inv.Totals.AnomalyCount = len(s.inv.Anomalies)
	s.inv.Totals.FindingCount = len(s.inv.ReferentialFindings)
	s.inv.Totals.ContentHash = "sha256:" + hashLines(s.allLines)
}

func (s *scanner) appendPrimary(class string, rec ObjectRecord) {
	s.primary[class] = append(s.primary[class], rec)
}

func (s *scanner) bumpStatus(class, status string) {
	if s.byStatus[class] == nil {
		s.byStatus[class] = map[string]int{}
	}
	s.byStatus[class][status]++
}

func (s *scanner) bumpKind(class, kind string) {
	if s.byKind[class] == nil {
		s.byKind[class] = map[string]int{}
	}
	s.byKind[class][kind]++
}

func (s *scanner) addAnomaly(t, rel, detail string) {
	s.inv.Anomalies = append(s.inv.Anomalies, Anomaly{Type: t, RelPath: rel, Detail: detail})
}

func (s *scanner) addFinding(t, from, to, detail string) {
	s.inv.ReferentialFindings = append(s.inv.ReferentialFindings, Finding{Type: t, From: from, To: to, Detail: detail})
}

func classKindOf(c string) string {
	if k, ok := classKind[c]; ok {
		return k
	}
	return "artifact"
}
