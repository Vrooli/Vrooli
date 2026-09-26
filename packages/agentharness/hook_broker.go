package agentharness

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// ManagedHookMarker is the single ownership marker used by every Vrooli hook
// writer. Hook IDs remain separate metadata so multiple Vrooli hooks can share
// one event without making the ownership decision ambiguous.
const ManagedHookMarker = "vrooli"

// HookTarget identifies one native JSON hook document. The broker owns the
// read-modify-write transaction for the target; resource adapters only choose
// the native path and event names.
type HookTarget struct {
	Agent string `json:"agent"`
	Path  string `json:"path"`
}

type HookRegistration struct {
	// Matcher is the native event-group matcher. Empty uses the broker's
	// catch-all matcher. Group is used by native formats, such as Claude's,
	// where ownership belongs to the event group rather than its inner
	// command entry.
	Matcher string
	Group   map[string]any
	Event   string         `json:"event"`
	ID      string         `json:"id"`
	Hook    map[string]any `json:"hook"`
}

type HookRecord struct {
	Agent string         `json:"agent"`
	Path  string         `json:"path"`
	Event string         `json:"event"`
	ID    string         `json:"id"`
	Hook  map[string]any `json:"hook"`
}

type HookResult struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Reason string `json:"reason"`
	Agent  string `json:"agent,omitempty"`
	Path   string `json:"path,omitempty"`
	Event  string `json:"event,omitempty"`
	ID     string `json:"id,omitempty"`
}

// HookBroker serializes hook mutations across processes with a lock file and
// publishes each document through an atomic rename. A broker has no global
// mutable state, so independent CLIs can safely construct one per command.
type HookBroker struct {
	LockTimeout time.Duration
}

func NewHookBroker() *HookBroker {
	return &HookBroker{LockTimeout: 5 * time.Second}
}

func (b *HookBroker) Reconcile(target HookTarget, registration HookRegistration) (HookResult, error) {
	result := hookResult(target, registration)
	if err := validateHookRequest(target, registration); err != nil {
		result.Status, result.Code, result.Reason = "failed", "invalid_arguments", err.Error()
		return result, err
	}
	var changed bool
	err := b.withDocumentLock(target.Path, func(document map[string]any) error {
		groups := eventGroups(document, registration.Event)
		if registration.Group != nil {
			desired := normalizeGroup(registration)
			found := false
			updated := make([]any, 0, len(groups)+1)
			for _, raw := range groups {
				group, ok := raw.(map[string]any)
				if !ok || groupIdentifier(group) != registration.ID {
					updated = append(updated, raw)
					continue
				}
				if !found && reflect.DeepEqual(group, desired) {
					updated = append(updated, raw)
				} else if !found {
					updated = append(updated, desired)
					changed = true
				} else {
					changed = true
				}
				found = true
			}
			if !found {
				updated = append(updated, desired)
				changed = true
			}
			setEventGroups(document, registration.Event, updated)
			return nil
		}

		desired := normalizeHook(registration)
		found := false
		for groupIndex, group := range groups {
			groupMap, ok := group.(map[string]any)
			if !ok {
				continue
			}
			entries, ok := hookEntries(groupMap)
			if !ok {
				continue
			}
			if groupIdentifier(groupMap) == registration.ID && isManagedGroup(groupMap) {
				updated := map[string]any{"matcher": groupMap["matcher"], "hooks": []any{desired}}
				if updated["matcher"] == nil {
					updated["matcher"] = "*"
				}
				if found {
					changed = true
					groups[groupIndex] = map[string]any{"matcher": "*", "hooks": []any{}}
				} else {
					found = true
					changed = !reflect.DeepEqual(groupMap, updated)
					groups[groupIndex] = updated
				}
				continue
			}
			updated := make([]any, 0, len(entries)+1)
			for _, raw := range entries {
				entry, ok := raw.(map[string]any)
				if !ok || hookIdentifier(entry) != registration.ID {
					updated = append(updated, raw)
					continue
				}
				if !found && reflect.DeepEqual(entry, desired) {
					updated = append(updated, raw)
				} else if !found {
					updated = append(updated, desired)
					changed = true
				} else {
					changed = true
				}
				found = true
			}
			if found {
				setHookEntries(groupMap, updated)
				groups[groupIndex] = groupMap
			}
		}
		if !found {
			groups = append(groups, map[string]any{
				"matcher": registration.Matcher,
				"hooks":   []any{desired},
			})
			if registration.Matcher == "" {
				groups[len(groups)-1].(map[string]any)["matcher"] = "*"
			}
			changed = true
		}
		setEventGroups(document, registration.Event, groups)
		return nil
	})
	if err != nil {
		result.Status, result.Code, result.Reason = "failed", "hook_write_failed", err.Error()
		return result, err
	}
	if !changed {
		result.Status, result.Code, result.Reason = "unchanged", "hook_unchanged", "hook is already configured"
		return result, nil
	}
	result.Status, result.Code, result.Reason = "applied", "hook_reconciled", "hook was written by the Vrooli hook broker"
	return result, nil
}

func (b *HookBroker) Remove(target HookTarget, event, id string) (HookResult, error) {
	registration := HookRegistration{Event: event, ID: id, Hook: map[string]any{}}
	result := hookResult(target, registration)
	if err := validateHookRequest(target, registration); err != nil {
		result.Status, result.Code, result.Reason = "failed", "invalid_arguments", err.Error()
		return result, err
	}
	var changed bool
	err := b.withDocumentLock(target.Path, func(document map[string]any) error {
		groups := eventGroups(document, event)
		updatedGroups := make([]any, 0, len(groups))
		for _, rawGroup := range groups {
			group, ok := rawGroup.(map[string]any)
			if !ok {
				updatedGroups = append(updatedGroups, rawGroup)
				continue
			}
			if groupIdentifier(group) == id {
				changed = true
				continue
			}
			entries, ok := hookEntries(group)
			if !ok {
				updatedGroups = append(updatedGroups, rawGroup)
				continue
			}
			kept := make([]any, 0, len(entries))
			for _, entry := range entries {
				m, isMap := entry.(map[string]any)
				if isMap && hookIdentifier(m) == id {
					changed = true
					continue
				}
				kept = append(kept, entry)
			}
			if len(kept) == 0 && len(kept) != len(entries) {
				continue
			}
			setHookEntries(group, kept)
			updatedGroups = append(updatedGroups, group)
		}
		if len(updatedGroups) == 0 {
			removeEvent(document, event)
		} else {
			setEventGroups(document, event, updatedGroups)
		}
		return nil
	})
	if err != nil {
		result.Status, result.Code, result.Reason = "failed", "hook_write_failed", err.Error()
		return result, err
	}
	if !changed {
		result.Status, result.Code, result.Reason = "unchanged", "hook_absent", "hook was already absent"
		return result, nil
	}
	result.Status, result.Code, result.Reason = "removed", "hook_removed", "hook was removed by the Vrooli hook broker"
	return result, nil
}

// List enumerates managed entries in all supplied targets. Missing native
// files are normal and are skipped; malformed files are returned as errors so
// callers cannot mistake an unreadable hook file for an empty registry.
func (b *HookBroker) List(targets []HookTarget) ([]HookRecord, error) {
	result := make([]HookRecord, 0)
	for _, target := range targets {
		if strings.TrimSpace(target.Path) == "" {
			continue
		}
		var document map[string]any
		err := b.withDocumentLock(target.Path, func(loaded map[string]any) error {
			document = loaded
			return nil
		})
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for event, groups := range allEventGroups(document) {
			for _, rawGroup := range groups {
				group, ok := rawGroup.(map[string]any)
				if !ok {
					continue
				}
				if isManagedGroup(group) {
					result = append(result, HookRecord{Agent: target.Agent, Path: target.Path, Event: event, ID: groupIdentifier(group), Hook: cloneMap(group)})
					continue
				}
				entries, ok := hookEntries(group)
				if !ok {
					continue
				}
				for _, rawEntry := range entries {
					entry, ok := rawEntry.(map[string]any)
					if !ok || !isManagedHook(entry) {
						continue
					}
					result = append(result, HookRecord{Agent: target.Agent, Path: target.Path, Event: event, ID: hookIdentifier(entry), Hook: cloneMap(entry)})
				}
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Agent != result[j].Agent {
			return result[i].Agent < result[j].Agent
		}
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		if result[i].Event != result[j].Event {
			return result[i].Event < result[j].Event
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// Migrate normalizes legacy ownership markers in place. It recognizes the
// former `_id` and `managedBy: vrooli-*` forms, plus the old policy-runner
// command with no marker, and rewrites them with managedBy=vrooli.
func (b *HookBroker) Migrate(target HookTarget) (int, error) {
	if strings.TrimSpace(target.Path) == "" {
		return 0, errors.New("hook target path is required")
	}
	count := 0
	err := b.withDocumentLock(target.Path, func(document map[string]any) error {
		if rootMarker, ok := document["managedBy"].(string); ok && rootMarker != "" && rootMarker != ManagedHookMarker {
			for _, groups := range allEventGroups(document) {
				for _, rawGroup := range groups {
					group, ok := rawGroup.(map[string]any)
					if !ok {
						continue
					}
					entries, ok := hookEntries(group)
					if !ok {
						continue
					}
					for _, rawEntry := range entries {
						entry, ok := rawEntry.(map[string]any)
						if !ok || isManagedHook(entry) {
							continue
						}
						entry["managedBy"] = ManagedHookMarker
						entry["_id"] = rootMarker
						count++
					}
				}
			}
			delete(document, "managedBy")
		}
		for _, groups := range allEventGroups(document) {
			for _, rawGroup := range groups {
				group, ok := rawGroup.(map[string]any)
				if !ok {
					continue
				}
				if isManagedGroup(group) {
					if group["managedBy"] != ManagedHookMarker || group["_id"] == nil {
						id := groupIdentifier(group)
						if id == "" {
							id = "vrooli-policy-runner"
						}
						group["managedBy"] = ManagedHookMarker
						group["_id"] = id
						count++
					}
					continue
				}
				entries, ok := hookEntries(group)
				if !ok {
					continue
				}
				for _, rawEntry := range entries {
					entry, ok := rawEntry.(map[string]any)
					if !ok || !legacyHookCandidate(entry) {
						continue
					}
					if entry["managedBy"] == ManagedHookMarker && entry["_id"] != nil {
						continue
					}
					id := hookIdentifier(entry)
					if id == "" {
						id = "vrooli-policy-runner"
					}
					entry["managedBy"] = ManagedHookMarker
					entry["_id"] = id
					count++
				}
			}
		}
		return nil
	})
	return count, err
}

func (b *HookBroker) withDocumentLock(path string, fn func(map[string]any) error) error {
	unlock, err := b.lockDocument(path)
	if err != nil {
		return err
	}
	defer unlock()
	document, err := readHookDocument(path)
	if err != nil {
		return err
	}
	before, _ := json.Marshal(document)
	if err := fn(document); err != nil {
		return err
	}
	after, err := json.Marshal(document)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(before, after) {
		return nil
	}
	if len(document) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	pretty, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteHookDocument(path, pretty)
}

// WithLock serializes a non-broker native projection with broker operations
// targeting the same JSON file. The callback must publish its own atomic
// document; the broker only owns the lock in this method.
func (b *HookBroker) WithLock(path string, fn func() error) error {
	unlock, err := b.lockDocument(path)
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func (b *HookBroker) lockDocument(path string) (func(), error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("hook target path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	lockPath := path + ".vrooli-lock"
	timeout := b.LockTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = lock.Close()
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for hook lock %s", lockPath)
		}
		time.Sleep(5 * time.Millisecond)
	}
	return func() { _ = os.Remove(lockPath) }, nil
}

func readHookDocument(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse hook document %s: %w", path, err)
	}
	return document, nil
}

func atomicWriteHookDocument(path string, data []byte) error {
	data = append(data, '\n')
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp := fmt.Sprintf("%s.vrooli-tmp-%d-%d", path, os.Getpid(), time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func validateHookRequest(target HookTarget, registration HookRegistration) error {
	if strings.TrimSpace(target.Agent) == "" {
		return errors.New("hook target agent is required")
	}
	if strings.TrimSpace(target.Path) == "" {
		return errors.New("hook target path is required")
	}
	if strings.TrimSpace(registration.Event) == "" || strings.TrimSpace(registration.ID) == "" || registration.Hook == nil {
		return errors.New("event, id, and hook are required")
	}
	return nil
}

func hookResult(target HookTarget, registration HookRegistration) HookResult {
	return HookResult{Agent: target.Agent, Path: target.Path, Event: registration.Event, ID: registration.ID}
}

func normalizeHook(registration HookRegistration) map[string]any {
	result := cloneMap(registration.Hook)
	delete(result, "hookId")
	delete(result, "id")
	result["_id"] = registration.ID
	result["managedBy"] = ManagedHookMarker
	return result
}

func normalizeGroup(registration HookRegistration) map[string]any {
	group := cloneMap(registration.Group)
	if registration.Matcher != "" {
		group["matcher"] = registration.Matcher
	} else if _, ok := group["matcher"]; !ok {
		group["matcher"] = "*"
	}
	group["managedBy"] = ManagedHookMarker
	group["_id"] = registration.ID
	group["hooks"] = []any{cloneMap(registration.Hook)}
	return group
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func eventGroups(document map[string]any, event string) []any {
	all, _ := document["hooks"].(map[string]any)
	groups, _ := all[event].([]any)
	return groups
}

func allEventGroups(document map[string]any) map[string][]any {
	result := map[string][]any{}
	all, _ := document["hooks"].(map[string]any)
	for event, raw := range all {
		if groups, ok := raw.([]any); ok {
			result[event] = groups
		}
	}
	return result
}

func setEventGroups(document map[string]any, event string, groups []any) {
	all, _ := document["hooks"].(map[string]any)
	if all == nil {
		all = map[string]any{}
		document["hooks"] = all
	}
	all[event] = groups
}

func removeEvent(document map[string]any, event string) {
	all, _ := document["hooks"].(map[string]any)
	if all == nil {
		return
	}
	delete(all, event)
	if len(all) == 0 {
		delete(document, "hooks")
	}
}

func hookEntries(group map[string]any) ([]any, bool) {
	entries, ok := group["hooks"].([]any)
	return entries, ok
}

func setHookEntries(group map[string]any, entries []any) {
	group["hooks"] = entries
}

func hookIdentifier(entry map[string]any) string {
	for _, key := range []string{"_id", "hookId", "id"} {
		if value, ok := entry[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	if value, ok := entry["managedBy"].(string); ok && value != "" && value != ManagedHookMarker {
		return value
	}
	return ""
}

func groupIdentifier(group map[string]any) string {
	for _, key := range []string{"_id", "hookId", "id"} {
		if value, ok := group[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	if value, ok := group["managedBy"].(string); ok && value != "" && value != ManagedHookMarker {
		return value
	}
	if entries, ok := hookEntries(group); ok {
		for _, raw := range entries {
			if entry, ok := raw.(map[string]any); ok {
				if id := hookIdentifier(entry); id != "" {
					return id
				}
			}
		}
	}
	if _, ok := group["managedBy"].(string); ok {
		return "vrooli-policy-runner"
	}
	return ""
}

func isManagedGroup(group map[string]any) bool {
	value, _ := group["managedBy"].(string)
	return value == ManagedHookMarker || strings.HasPrefix(value, ManagedHookMarker+"-")
}

func isManagedHook(entry map[string]any) bool {
	value, _ := entry["managedBy"].(string)
	return value == ManagedHookMarker || strings.HasPrefix(value, ManagedHookMarker+"-") || entry["_id"] != nil
}

func legacyHookCandidate(entry map[string]any) bool {
	if isManagedHook(entry) {
		return true
	}
	for _, key := range []string{"command", "script"} {
		if value, ok := entry[key].(string); ok && (strings.Contains(value, "vrooli-policy-runner") || strings.Contains(value, "pretooluse-bash-deny") || strings.Contains(value, "vrooli-bash-deny") || strings.Contains(value, "vrooli-memory hook")) {
			return true
		}
	}
	return false
}
