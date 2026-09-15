package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EffortDiscoveryConfig struct {
	Root                 string
	ScanLimit            int
	FileBytes            int64
	Interval             time.Duration
	StaleAfter           time.Duration
	StandingAllowanceRef string
}
type discoveredEffort struct {
	enrollment   *pb.EffortEnrollment
	observation  *pb.EffortBoardRow
	manifestless bool
}

// openSafeRoot anchors every directory handle and rejects symlinks, including in
// the configured root. Root handles prevent path substitution from escaping scope.
func openSafeRoot(path string) (*os.Root, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("effort root must be absolute")
	}
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	root, err := os.OpenRoot(volume + string(filepath.Separator))
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(clean, volume+string(filepath.Separator)), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		next, err := safeChildRoot(root, part)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = next
	}
	return root, nil
}
func safeChildRoot(root *os.Root, name string) (*os.Root, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("directory must not be a symlink")
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	actual, err := child.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		child.Close()
		return nil, errors.New("directory changed during open")
	}
	return child, nil
}
func safeRead(root *os.Root, name string, limit int64) ([]byte, error) {
	if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
		return nil, errors.New("source path escapes workspace")
	}
	parts := strings.Split(filepath.ToSlash(name), "/")
	at := root
	for _, part := range parts[:len(parts)-1] {
		next, err := safeChildRoot(at, part)
		if at != root {
			at.Close()
		}
		if err != nil {
			return nil, err
		}
		at = next
	}
	if at != root {
		defer at.Close()
	}
	base := parts[len(parts)-1]
	info, err := at.Lstat(base)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("source must be a bounded regular file")
	}
	file, err := at.Open(base)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return nil, errors.New("source changed during open")
	}
	after, err := at.Lstat(base)
	if err != nil || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, after) {
		return nil, errors.New("source changed during open")
	}
	b, err := io.ReadAll(io.LimitReader(file, limit+1))
	if int64(len(b)) > limit {
		return nil, errors.New("source exceeds byte limit")
	}
	return b, err
}

// normalizeLegacyDriver is a narrow temporary adapter, not a recursive source
// loader. Unknown fields are ignored and no declared path is followed. An absent
// version means legacy v0; version 1 has the same allowlisted observation shape.
func normalizeLegacyDriver(raw []byte, e *pb.EffortEnrollment, o *pb.EffortBoardRow, source string) error {
	var driver struct {
		Version      *int   `json:"schema_version"`
		Effort       string `json:"effort"`
		NextAction   string `json:"next_action"`
		LoopState    string `json:"loop_state"`
		Orchestrator struct {
			LoopState string `json:"loop_state"`
		} `json:"orchestrator"`
		Assignment string                     `json:"assignment"`
		Children   map[string]json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(raw, &driver); err != nil {
		return fmt.Errorf("legacy driver shape: %w", err)
	}
	if driver.Version != nil && *driver.Version != 1 {
		return errors.New("unsupported legacy driver schema version")
	}
	if strings.TrimSpace(driver.Effort) == "" || len(driver.Effort) > 480 ||
		(strings.TrimSpace(driver.NextAction) == "" && strings.TrimSpace(driver.LoopState) == "" && strings.TrimSpace(driver.Orchestrator.LoopState) == "") {
		return errors.New("legacy driver requires bounded effort string and next_action or loop_state")
	}
	if len(driver.NextAction) > 4096 || len(driver.LoopState) > 1024 || len(driver.Orchestrator.LoopState) > 1024 || len(driver.Assignment) > 2048 || len(driver.Children) > 100 {
		return errors.New("legacy driver observation exceeds field or 100-child limit")
	}
	if e.EffortRef == "" {
		e.EffortRef = "legacy-effort:" + driver.Effort
	}
	if e.DisplayName == "" {
		e.DisplayName = driver.Effort
	}
	if o.NextAction == "" {
		o.NextAction = driver.NextAction
	}
	loops := []string{}
	if driver.LoopState != "" {
		loops = append(loops, "loop_state="+driver.LoopState)
	}
	if driver.Orchestrator.LoopState != "" {
		loops = append(loops, "orchestrator.loop_state="+driver.Orchestrator.LoopState)
	}
	if len(loops) > 0 {
		o.Rationale = strings.TrimSpace(o.Rationale + " Legacy driver self-report: " + strings.Join(loops, "; "))
	}
	o.EvidenceRefs = append(o.EvidenceRefs, "workspace:"+e.Workspace+"/"+source+"@"+bytesDigest(raw))
	if driver.Assignment != "" {
		o.EvidenceRefs = append(o.EvidenceRefs, "declared-assignment:"+driver.Assignment)
	}
	o.Limitations = append(o.Limitations, "legacy driver observations and child references are self-report; status, attempt and execution_id are not owner runtime or cost evidence")
	keys := make([]string, 0, len(driver.Children))
	for key := range driver.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	seen := map[string]bool{}
	for _, sub := range e.Subjects {
		seen[sub.RunId] = true
	}
	for _, key := range keys {
		var child struct {
			RunID string `json:"run_id"`
		}
		if err := json.Unmarshal(driver.Children[key], &child); err != nil {
			o.Limitations = append(o.Limitations, "legacy child has incompatible run reference")
			continue
		}
		id, err := uuid.Parse(child.RunID)
		if err != nil || id == uuid.Nil {
			o.Limitations = append(o.Limitations, "legacy child current AM run reference is unknown or invalid")
			continue
		}
		if seen[id.String()] {
			continue
		}
		if len(e.Subjects) >= 100 {
			o.Limitations = append(o.Limitations, "legacy subjects capped at 100 including manifest subjects")
			break
		}
		seen[id.String()] = true
		e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: id.String(), RunId: id.String(), Role: "worker", Assignment: "legacy driver child (self-report)"})
	}
	return nil
}

// Schema 1 is the canonical workspace helper's contract. Additional unrelated
// manifest fields are ignored; only declared bounded supervision sources are read.
// Resolution sources are change signals, never an executable instruction or grant.
func observeResolutionSources(folder *os.Root, e *pb.EffortEnrollment, o *pb.EffortBoardRow, sources []string, limit int64) error {
	if len(sources) > 8 {
		return errors.New("observation_sources exceeds eight-source limit")
	}
	const legacyAnswers = "handoffs/OPERATOR-ANSWERS.md"
	declared := map[string]bool{}
	for _, source := range sources {
		if len(source) > 512 || !filepath.IsLocal(source) || filepath.ToSlash(filepath.Clean(source)) != source || strings.Contains(source, "\\") ||
			(!strings.HasPrefix(source, "handoffs/") && !strings.HasPrefix(source, "evidence/") && !strings.HasPrefix(source, "findings/")) {
			return errors.New("observation source must be a clean relative path under handoffs/, evidence/ or findings/")
		}
		if declared[source] {
			return errors.New("duplicate observation source")
		}
		declared[source] = true
	}
	paths := append([]string{}, sources...)
	if !declared[legacyAnswers] {
		paths = append(paths, legacyAnswers)
	}
	sort.Strings(paths)
	for _, source := range paths {
		b, err := safeRead(folder, source, limit)
		if err != nil {
			if !declared[source] && errors.Is(err, os.ErrNotExist) {
				continue
			}
			o.Freshness = pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE
			o.Limitations = append(o.Limitations, "resolution source unavailable: "+source)
			continue
		}
		o.EvidenceRefs = append(o.EvidenceRefs, "workspace:"+e.Workspace+"/"+source+"@"+bytesDigest(b))
		o.Limitations = append(o.Limitations, "resolution source "+source+" is a bounded change signal; verify its owner receipt and recovery authority")
	}
	return nil
}

func readWorkspace(root *os.Root, name string, limit int64, now time.Time) (*discoveredEffort, error) {
	folder, err := safeChildRoot(root, name)
	if err != nil {
		return nil, err
	}
	defer folder.Close()
	raw, err := safeRead(folder, "effort.json", limit)
	if errors.Is(err, os.ErrNotExist) {
		legacy, readErr := safeRead(folder, "handoffs/state.json", limit)
		if readErr != nil {
			return nil, fmt.Errorf("manifest absent; legacy checkpoint unavailable: %w", readErr)
		}
		e := &pb.EffortEnrollment{Workspace: name, SourceRevision: bytesDigest(legacy)}
		o := &pb.EffortBoardRow{ObservedAt: timestamppb.New(now), Freshness: pb.EffortFreshness_EFFORT_FRESHNESS_FRESH, RuntimeState: "unknown", OutcomeStanding: &pb.EffortOutcomeStanding{State: "unknown", Attribution: "legacy driver self-report"}, Limitations: []string{"manifest absent; provisional legacy identity", "accepted destination, target revision and actual owner grant are unknown"}}
		if err = normalizeLegacyDriver(legacy, e, o, "handoffs/state.json"); err != nil {
			return nil, err
		}
		if err = observeResolutionSources(folder, e, o, nil, limit); err != nil {
			return nil, err
		}
		return &discoveredEffort{enrollment: e, observation: o, manifestless: true}, nil
	}
	if err != nil {
		return nil, err
	}
	var m struct {
		SchemaVersion  int    `json:"schema_version"`
		Slug           string `json:"slug"`
		Repository     string `json:"repository"`
		EffortRef      string `json:"effort_ref"`
		EffortID       string `json:"effort_id"`
		DisplayName    string `json:"display_name"`
		DestinationRef string `json:"destination_ref"`
		TargetRevision string `json:"target_revision"`
		WorkShape      string `json:"work_shape"`
		Stage          string `json:"stage"`
		Execution      struct {
			Status               string `json:"status"`
			ApprovalRef          string `json:"approval_ref"`
			ApprovedSourceDigest string `json:"approved_source_digest"`
		} `json:"execution"`
		Owners             map[string]json.RawMessage `json:"owners"`
		Supervision        json.RawMessage            `json:"supervision"`
		Enrollment         json.RawMessage            `json:"enrollment"`
		Checkpoint         string                     `json:"checkpoint"`
		ObservationSources []string                   `json:"observation_sources"`
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("malformed effort manifest: %w", err)
	}
	e := &pb.EffortEnrollment{}
	switch m.SchemaVersion {
	case 1:
		if m.Slug == "" || m.Repository == "" {
			return nil, errors.New("schema 1 requires slug and repository")
		}
		e.EffortRef = m.EffortRef
		if e.EffortRef == "" {
			e.EffortRef = m.EffortID
		}
		if e.EffortRef == "" {
			e.EffortRef = "workspace:" + m.Repository + "#" + m.Slug
		}
		e.DisplayName = m.DisplayName
		if e.DisplayName == "" {
			e.DisplayName = m.Slug
		}
		e.DestinationRef = m.DestinationRef
		e.TargetRevision = m.TargetRevision
		if e.TargetRevision == "" {
			e.TargetRevision = m.Execution.ApprovedSourceDigest
		}
		e.AuthorityRef = m.Execution.ApprovalRef
		e.WorkShape = m.WorkShape
		if len(m.Supervision) > 0 {
			declared := &pb.EffortEnrollment{}
			if err = protojson.Unmarshal(m.Supervision, declared); err != nil {
				return nil, fmt.Errorf("incompatible supervision declaration: %w", err)
			}
			e.SupervisorRunId = declared.SupervisorRunId
			e.Subjects = declared.Subjects
		}
		// Canonical owner keys carry exact references; no effort-specific layouts.
		for _, role := range []string{"orchestrator", "worker", "supervisor", "reviewer", "investigator", "repair"} {
			if ownerRaw, ok := m.Owners[role]; ok {
				sub := &pb.EffortSubject{}
				if err = protojson.Unmarshal(ownerRaw, sub); err != nil {
					return nil, fmt.Errorf("owner %s must be a typed subject: %w", role, err)
				}
				sub.Role = role
				e.Subjects = append(e.Subjects, sub)
			}
			var id string
			if ownerRaw, ok := m.Owners[role+"_run_id"]; ok {
				if err = json.Unmarshal(ownerRaw, &id); err != nil {
					return nil, err
				}
				e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: id, RunId: id, Role: role})
			}
		}
	case 2:
		if err = protojson.Unmarshal(m.Enrollment, e); err != nil {
			return nil, fmt.Errorf("incompatible schema 2 enrollment: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported effort schema version %d", m.SchemaVersion)
	}
	e.Workspace = name
	e.SourceRevision = bytesDigest(raw)
	// A file's claim of permission is not an authenticated owner grant.
	e.PermittedActions = nil
	e.SupervisorOwnerSubject = ""
	e.SupervisorScope = ""
	e.DispatchAuthorization = nil
	e.AuthorizedBy = ""
	e.AuthorityExpiresAt = nil
	e.MaximumDirectives = 0
	e.Withdrawn = false
	e.Revision = 0
	if err = validateEffortEnrollment(e, false, now); err != nil {
		return nil, err
	}
	o := &pb.EffortBoardRow{ObservedAt: timestamppb.New(now), Freshness: pb.EffortFreshness_EFFORT_FRESHNESS_FRESH, RuntimeState: "unknown", OutcomeStanding: &pb.EffortOutcomeStanding{State: "unknown", Attribution: "workspace self-report"}, EvidenceRefs: []string{"workspace:" + name + "/effort.json@" + e.SourceRevision}, Limitations: []string{"workspace declarations are attributed self-report; authority is unverified"}}
	if e.DestinationRef == "" {
		o.Limitations = append(o.Limitations, "accepted destination reference is missing")
	}
	if e.TargetRevision == "" {
		o.Limitations = append(o.Limitations, "accepted target revision is missing; source digest is not an acceptance revision")
	}
	if m.Checkpoint != "" {
		// Only a declared checkpoint beneath handoffs or evidence is eligible.
		if !strings.HasPrefix(m.Checkpoint, "handoffs/") && !strings.HasPrefix(m.Checkpoint, "evidence/") {
			return nil, errors.New("checkpoint must be under handoffs/ or evidence/")
		}
		b, readErr := safeRead(folder, m.Checkpoint, limit)
		if readErr != nil {
			return nil, fmt.Errorf("checkpoint: %w", readErr)
		}
		var cut struct {
			Effort     json.RawMessage `json:"effort"`
			NextAction string          `json:"next_action"`
			Rationale  string          `json:"rationale"`
			Pending    []string        `json:"pending_operations"`
			Blockers   []string        `json:"blockers"`
		}
		if err = json.Unmarshal(b, &cut); err != nil {
			return nil, err
		}
		o.NextAction = cut.NextAction
		o.Rationale = cut.Rationale
		o.PendingOperations = cut.Pending
		o.Blockers = cut.Blockers
		if len(cut.Effort) > 0 {
			if err = normalizeLegacyDriver(b, e, o, m.Checkpoint); err != nil {
				return nil, fmt.Errorf("declared driver checkpoint: %w", err)
			}
		} else {
			o.EvidenceRefs = append(o.EvidenceRefs, "workspace:"+name+"/"+m.Checkpoint+"@"+bytesDigest(b))
		}
	} else {
		// This one fixed fallback is authorized for incomplete canonical manifests.
		// A declared checkpoint above always wins, including when it is unavailable.
		b, readErr := safeRead(folder, "handoffs/state.json", limit)
		if readErr == nil {
			readErr = normalizeLegacyDriver(b, e, o, "handoffs/state.json")
		}
		if readErr != nil {
			o.Limitations = append(o.Limitations, "legacy checkpoint unavailable: "+readErr.Error())
			if !errors.Is(readErr, os.ErrNotExist) {
				o.Freshness = pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE
			}
		}
	}
	if len(e.Subjects) == 0 {
		o.Limitations = append(o.Limitations, "no typed current owner run references declared")
	}
	if err = observeResolutionSources(folder, e, o, m.ObservationSources, limit); err != nil {
		return nil, err
	}
	if b, readErr := safeRead(folder, "requirements.json", limit); readErr == nil {
		var rows []struct {
			Assessment string            `json:"assessment"`
			Evidence   []json.RawMessage `json:"evidence"`
		}
		if err = json.Unmarshal(b, &rows); err != nil {
			return nil, fmt.Errorf("requirements: %w", err)
		}
		count, met := uint32(len(rows)), uint32(0)
		for _, row := range rows {
			if row.Assessment == "met" && len(row.Evidence) > 0 {
				met++
			}
		}
		o.OutcomeStanding.RequiredCount = &count
		o.OutcomeStanding.MetCount = &met
		o.OutcomeStanding.State = "unverified"
		o.OutcomeStanding.Limitations = []string{"requirement claims are not independent owner acceptance"}
		o.OutcomeStanding.EvidenceRefs = []string{"workspace:" + name + "/requirements.json@" + bytesDigest(b)}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return nil, fmt.Errorf("requirements: %w", readErr)
	}
	if m.Stage == "complete" || m.Stage == "completed" || m.Stage == "closed" {
		o.OutcomeStanding.State = "reported_complete"
		o.Limitations = append(o.Limitations, "reported completion does not retire supervision without an owner disposition")
	}
	o.ChangeIdentity = effortDigest(o)
	return &discoveredEffort{enrollment: e, observation: o}, nil
}

// ReconcileDiscovery is an explicit mutation called by the existing scheduler.
// A failed or capped scan preserves prior enrollments and cannot infer closure.
func (s *EffortService) ReconcileDiscovery(ctx context.Context) (*pb.EffortDiscovery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconcileDiscovery(ctx)
}
func (s *EffortService) reconcileDiscovery(ctx context.Context) (*pb.EffortDiscovery, error) {
	now := s.now().UTC()
	previous, err := s.repo.GetEffortDiscovery(ctx)
	if err != nil {
		return nil, err
	}
	// Reuse a retained canonical identity when only its manifest disappeared.
	// This is a bounded owner read, not identity inferred from a directory name.
	existing, err := s.repo.ListEfforts(ctx, "", 1001)
	if err != nil {
		return nil, err
	}
	byWorkspace := map[string][]*pb.EffortEnrollment{}
	for _, e := range existing {
		if e.Workspace != "" {
			byWorkspace[e.Workspace] = append(byWorkspace[e.Workspace], e)
		}
	}
	d := &pb.EffortDiscovery{Generation: previous.Generation, LastScanAt: timestamppb.New(now), LastSuccessfulScanAt: previous.LastSuccessfulScanAt, ScanLimit: uint32(s.config.ScanLimit), ByteLimit: uint64(s.config.FileBytes), Root: s.config.Root}
	d.DirectoryLimit = 1000
	d.StandingAllowanceRef = s.config.StandingAllowanceRef
	d.StandingUsage = &pb.EffortUsage{Partial: true, Source: "standing effort discovery", Limitations: []string{"standing discovery and shared idle cost are unmeasured; no allowance is inferred"}}
	d.ScanCursor = previous.GetScanCursor()
	paths := map[string]bool{}
	invalidPaths := map[string]bool{}
	completeIndex := false
	// Retain failures from pages not revisited in this scan.
	for _, f := range previous.Findings {
		if f.Code != "not_scanned" && f.Source != "workspace-root" && f.Source != "enrollments" && f.Source != "agent-manager:runs" {
			d.Findings = append(d.Findings, f)
		}
	}
	finding := func(source, code string, err error) {
		d.Partial = true
		d.Findings = append(d.Findings, &pb.EffortDiscoveryFinding{Source: source, Code: code, Reason: err.Error()})
	}
	root, err := openSafeRoot(s.config.Root)
	found := map[string]*discoveredEffort{}
	conflicts := map[string]bool{}
	if err != nil {
		finding("workspace-root", "unavailable", err)
	} else {
		defer root.Close()
		dir, err := root.Open(".")
		if err != nil {
			finding("workspace-root", "unavailable", err)
		} else {
			entries, readErr := dir.ReadDir(int(d.DirectoryLimit) + 1)
			dir.Close()
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				finding("workspace-root", "read_failed", readErr)
			}
			if len(entries) > int(d.DirectoryLimit) {
				finding("workspace-root", "directory_limit", errors.New("root exceeds 1000-entry index limit; scan refused without inferring removal"))
				entries = nil
			} else if readErr == nil || errors.Is(readErr, io.EOF) {
				completeIndex = true
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
			for _, entry := range entries {
				paths[entry.Name()] = true
			}
			start := sort.Search(len(entries), func(i int) bool { return entries[i].Name() > previous.GetScanCursor() })
			if start == len(entries) {
				start = 0
			}
			if len(entries) > s.config.ScanLimit {
				rotated := append(append([]os.DirEntry{}, entries[start:]...), entries[:start]...)
				entries = rotated[:s.config.ScanLimit]
				d.Partial = true
			}
			for _, entry := range entries {
				if err = ctx.Err(); err != nil {
					return nil, err
				}
				d.ScannedCount++
				d.ScanCursor = entry.Name()
				retained := d.Findings[:0]
				for _, f := range d.Findings {
					if f.Source != entry.Name() {
						retained = append(retained, f)
					}
				}
				d.Findings = retained
				if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
					continue
				}
				item, readErr := readWorkspace(root, entry.Name(), s.config.FileBytes, now)
				if readErr != nil {
					invalidPaths[entry.Name()] = true
					finding(entry.Name(), "invalid_manifest", readErr)
					continue
				}
				if item.manifestless {
					prior := byWorkspace[entry.Name()]
					if len(prior) > 1 {
						invalidPaths[entry.Name()] = true
						finding(entry.Name(), "identity_conflict", errors.New("manifestless workspace has multiple retained identities"))
						continue
					}
					if len(prior) == 1 {
						item.enrollment.EffortRef = prior[0].EffortRef
					}
				}
				if item.observation.Freshness == pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
					finding(entry.Name(), "checkpoint_unavailable", errors.New("checkpoint or resolution evidence unavailable; source observation retained with uncertainty"))
				}
				ref := item.enrollment.EffortRef
				if _, ok := found[ref]; ok {
					conflicts[ref] = true
					finding(entry.Name(), "identity_conflict", fmt.Errorf("duplicate effort identity %q", ref))
				}
				found[ref] = item
			}
		}
	}
	refs := make([]string, 0, len(found))
	for ref := range found {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	identities := []string{}
	for _, ref := range refs {
		item := found[ref]
		if conflicts[ref] {
			continue
		}
		e, o := item.enrollment, item.observation
		old, oldO, getErr := s.repo.GetEffort(ctx, ref)
		if getErr != nil && !errors.Is(getErr, ErrNotFound) {
			return nil, getErr
		}
		if old != nil && old.Workspace != "" && old.Workspace != e.Workspace && paths[old.Workspace] {
			conflicts[ref] = true
			finding(e.Workspace, "identity_conflict", fmt.Errorf("identity %q also belongs to present workspace %q", ref, old.Workspace))
			continue
		}
		// Loss of a previously observed fallback is unavailable evidence, even if
		// the manifest itself still reads successfully. Do not label that cut fresh.
		if oldO != nil {
			prefix := "workspace:" + e.Workspace + "/handoffs/state.json@"
			had, has := false, false
			for _, ref := range oldO.EvidenceRefs {
				had = had || strings.HasPrefix(ref, prefix)
			}
			for _, ref := range o.EvidenceRefs {
				has = has || strings.HasPrefix(ref, prefix)
			}
			if had && !has && strings.Contains(strings.Join(o.Limitations, " "), "legacy checkpoint unavailable") {
				o.Freshness = pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE
				// Retain the last observed source identity so repeated failed scans
				// cannot forget the dependency and manufacture a fresh manifest-only cut.
				o.ObservedAt = oldO.ObservedAt
				for _, ref := range oldO.EvidenceRefs {
					if strings.HasPrefix(ref, prefix) {
						o.EvidenceRefs = append(o.EvidenceRefs, ref)
					}
				}
				finding(e.Workspace, "checkpoint_unavailable", errors.New("previously observed legacy checkpoint is unavailable"))
			}
		}
		// Observation identity excludes collection time; stable cuts do not increment revisions.
		cut := proto.Clone(o).(*pb.EffortBoardRow)
		cut.ObservedAt = nil
		cut.ChangeIdentity = ""
		identity := effortDigest(e) + effortDigest(cut)
		identities = append(identities, ref+":"+identity)
		expected := uint64(0)
		if old != nil {
			expected = old.Revision
			e.Withdrawn = old.Withdrawn
			e.WithdrawalReason = old.WithdrawalReason
			// Retain a verified grant only across an unchanged target and exact subject binding.
			if old.TargetRevision == e.TargetRevision && sameEffortSubjects(old, e) {
				e.AuthorizedBy = old.AuthorizedBy
				e.PermittedActions = old.PermittedActions
				e.AuthorityExpiresAt = old.AuthorityExpiresAt
				e.MaximumDirectives = old.MaximumDirectives
				e.CooldownSeconds = old.CooldownSeconds
				e.SupervisorRunId = old.SupervisorRunId
				e.SupervisorOwnerSubject = old.SupervisorOwnerSubject
				e.SupervisorScope = old.SupervisorScope
				e.DispatchAuthorization = old.DispatchAuthorization
			}
			if oldO.GetChangeIdentity() == identity && old.Workspace == e.Workspace {
				oldO.ObservedAt = timestamppb.New(now)
				if err = s.repo.RefreshEffortObservation(ctx, ref, old.Revision, oldO); err != nil {
					return nil, err
				}
				continue
			}
		}
		e.Revision = expected + 1
		e.UpdatedAt = timestamppb.New(now)
		o.ChangeIdentity = identity
		if err = s.repo.SaveEffort(ctx, e, o, expected, fmt.Sprintf("discovery:%s:%d", ref, e.Revision), identity); err != nil {
			return nil, err
		}
	}
	// Bound reconciliation independently of discovery; never scan an unbounded registry.
	if len(existing) > 1000 {
		finding("enrollments", "registry_limit", errors.New("enrollment reconciliation capped at 1000"))
		existing = existing[:1000]
	}
	for _, e := range existing {
		if e.Workspace == "" {
			continue
		}
		_, seen := found[e.EffortRef]
		if seen && !conflicts[e.EffortRef] {
			continue
		}
		if !conflicts[e.EffortRef] && !invalidPaths[e.Workspace] && (!completeIndex || paths[e.Workspace]) {
			continue
		}
		_, o, getErr := s.repo.GetEffort(ctx, e.EffortRef)
		if getErr != nil {
			return nil, getErr
		}
		if o.Freshness == pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
			continue
		}
		expected := e.Revision
		e.Revision++
		e.UpdatedAt = timestamppb.New(now)
		o.Freshness = pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE
		o.Limitations = append(o.Limitations, "workspace removed or identity conflicted; outcome remains unverified")
		o.ChangeIdentity = "unavailable:" + o.ChangeIdentity
		if err = s.repo.SaveEffort(ctx, e, o, expected, fmt.Sprintf("discovery:%s:%d", e.EffortRef, e.Revision), o.ChangeIdentity); err != nil {
			return nil, err
		}
	}
	if registryErr := s.reconcileRunRegistry(ctx, d); registryErr != nil {
		finding("agent-manager:runs", "registry_unavailable", registryErr)
	}
	// Hash the complete bounded retained cut rather than only this rotating page.
	identities = nil
	current, err := s.repo.ListEfforts(ctx, "", 1001)
	if err != nil {
		return nil, err
	}
	for _, e := range current {
		_, o, getErr := s.repo.GetEffort(ctx, e.EffortRef)
		if getErr != nil {
			return nil, getErr
		}
		identities = append(identities, e.EffortRef+":"+o.ChangeIdentity+":"+fmt.Sprint(e.Revision))
	}
	if len(d.Findings) > 0 {
		d.Partial = true
	}
	sort.Slice(d.Findings, func(i, j int) bool {
		return d.Findings[i].Source+d.Findings[i].Code < d.Findings[j].Source+d.Findings[j].Code
	})
	digestCut := proto.Clone(d).(*pb.EffortDiscovery)
	digestCut.Generation = 0
	digestCut.LastScanAt = nil
	digestCut.LastSuccessfulScanAt = nil
	digestCut.ScanCursor = ""
	digestCut.ScannedCount = 0
	digestCut.Partial = false // page position is coverage, not a changed evidence cut
	d.ChangeIdentity = bytesDigest([]byte(strings.Join(identities, "\n") + effortDigest(digestCut)))
	if d.ChangeIdentity != previous.ChangeIdentity {
		d.Generation++
	}
	if completeIndex && len(d.Findings) == 0 {
		d.LastSuccessfulScanAt = timestamppb.New(now)
	}
	if err = s.repo.SaveEffortDiscovery(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func sameEffortSubjects(a, b *pb.EffortEnrollment) bool {
	return proto.Equal(&pb.EffortEnrollment{Subjects: a.Subjects}, &pb.EffortEnrollment{Subjects: b.Subjects})
}
