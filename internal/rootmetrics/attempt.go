package rootmetrics

// This file records observations only. No activation, recovery, or authorization
// decision may read these records. A missing or incomplete record is expected after
// a crash or an I/O error and must never be interpreted as success.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/statecontract"
)

const rootAttemptSchema = 1
const rootAttemptRetention = 90 * 24 * time.Hour

type rootAttemptEvent struct {
	At     time.Time `json:"at"`
	Stage  string    `json:"stage"`
	Reason string    `json:"reason,omitempty"`
}
type rootAttemptRecord struct {
	Schema            int                `json:"schema_version"`
	ID                string             `json:"id"`
	Kind              string             `json:"kind"`
	OperationID       string             `json:"operation_id,omitempty"`
	ServingContract   string             `json:"serving_contract,omitempty"`
	CandidateContract string             `json:"candidate_contract,omitempty"`
	Events            []rootAttemptEvent `json:"events"`
	Initiator         string             `json:"initiator,omitempty"`
	CandidateImage    string             `json:"candidate_image_sha256,omitempty"`
	BundleManifest    string             `json:"bundle_manifest_sha256,omitempty"`
	Images            map[string]string  `json:"images,omitempty"`
	Attribution       string             `json:"attribution,omitempty"`
	Owners            []string           `json:"owners,omitempty"`
}
type Attempt struct {
	mu     sync.Mutex
	path   string
	record rootAttemptRecord
	warn   sync.Once
	closed bool
	writes chan []byte
	done   chan struct{}
	write  func(string, []byte) error
}

// Async moves durable observation writes off safety-lock holders. Call Flush
// after releasing those locks; a crash can leave a partial observation.
func (a *Attempt) Async() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.writes != nil {
		return
	}
	a.writes = make(chan []byte, 65)
	a.done = make(chan struct{})
	writes, done := a.writes, a.done
	go func() {
		defer close(done)
		for data := range writes {
			// Each snapshot contains all preceding events. Coalesce queued
			// snapshots to avoid repeated fsyncs when a transition is busy.
		collect:
			for {
				select {
				case newer, ok := <-writes:
					if !ok {
						a.persist(data)
						return
					}
					data = newer
				default:
					break collect
				}
			}
			a.persist(data)
		}
	}()
}

// Flush gives queued observations a bounded opportunity to finish. It must run
// outside safety locks; process exit can still leave the record incomplete.
func (a *Attempt) Flush() {
	if a == nil {
		return
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	writes, done := a.writes, a.done
	if writes != nil {
		close(writes)
	}
	a.mu.Unlock()
	if writes == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		fmt.Fprintln(os.Stderr, "Warning: root operation observation remains pending; activation continues without a complete receipt.")
	}
}

// Begin is deliberately best effort. Its callers must not branch on
// the result. IDs are unique per invocation, including retries and resumptions.
func Begin(home, kind string) *Attempt {
	if kind != "install" && kind != "rebaseline" && kind != "activation_observation" && kind != "bundle_publication" {
		return nil
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: root operation observation unavailable; activation continues without a complete receipt.")
		return nil
	}
	dir := filepath.Join(state, "root-rebaseline", "attempts")
	if _, err = config.EnsureOwnedDirMode(dir, 0700); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: root operation observation unavailable; activation continues without a complete receipt.")
		return nil
	}
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: root operation observation unavailable; activation continues without a complete receipt.")
		return nil
	}
	now := time.Now().UTC()
	id := now.Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(token[:])
	a := &Attempt{path: filepath.Join(dir, id+".json"), record: rootAttemptRecord{Schema: rootAttemptSchema, ID: id, Kind: kind, Initiator: observeInitiator()}, write: writeObservation}
	a.Event("started", "")
	pruneRootAttempts(dir, now)
	return a
}

// CandidateImage ties an activation observation to the frozen executable.
func (a *Attempt) CandidateImage(hash string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.record.CandidateImage = rootAttemptHash(hash)
	a.mu.Unlock()
}

// Bundle binds forensic publication evidence to exact verified content. Only
// known image names and full hashes are accepted; paths never enter records.
func (a *Attempt) Bundle(manifest string, images map[string]string) {
	if a == nil || rootAttemptHash(manifest) == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.record.BundleManifest = manifest
	a.record.Images = make(map[string]string)
	for _, name := range []string{"vrooli", "vrooli-atomic-install", "vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-watchdog"} {
		if hash := rootAttemptHash(images[name]); hash != "" {
			a.record.Images[name] = hash
		}
	}
}

// Only fixed code-like tokens and full contract hashes are accepted; errors,
// source paths, and operator input are never written to the observation log.
func rootAttemptToken(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}
func rootAttemptHash(s string) string {
	if len(s) != 64 {
		return ""
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ""
		}
	}
	return s
}
func (a *Attempt) Identities(serving, candidate string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.record.ServingContract, a.record.CandidateContract = rootAttemptHash(serving), rootAttemptHash(candidate)
	a.mu.Unlock()
}

// Operation binds a coordinator attempt or outcome observation to the activation receipt.
func (a *Attempt) Operation(id string) {
	if a == nil || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(id) {
		return
	}
	a.mu.Lock()
	a.record.OperationID = id
	a.mu.Unlock()
}
func (a *Attempt) Event(stage, reason string) {
	if a == nil || !rootAttemptToken(stage) || reason != "" && !rootAttemptToken(reason) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || len(a.record.Events) >= 64 {
		return
	}
	a.record.Events = append(a.record.Events, rootAttemptEvent{At: time.Now().UTC(), Stage: stage, Reason: reason})
	b, err := json.Marshal(a.record)
	if err == nil && a.writes != nil {
		a.writes <- append(b, '\n')
		return
	}
	if err == nil {
		a.persist(append(b, '\n'))
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: root operation observation could not be durably recorded; activation continues without a complete receipt.")
	}
}

func (a *Attempt) persist(data []byte) {
	if err := a.write(a.path, data); err != nil {
		a.warn.Do(func() {
			fmt.Fprintln(os.Stderr, "Warning: root operation observation could not be durably recorded; activation continues without a complete receipt.")
		})
	}
}

// Each attempt has a random private path and a single writer. A per-target
// owned-write lock would leak one lock file per observation. Observations are
// best effort and never affect activation. A successful write confirms the
// file and complete logical and physical ancestor chain.
func writeObservation(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".observation-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		err = config.ChownToInvokingUser(tmp)
	}
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		return err
	}
	return config.SyncOwnedDirectoryChain(filepath.Dir(path))
}

// Retain observations for 90 days. Only our own regular JSON files are pruned;
// failures are ignored, and no receipt, pause marker, or transition is touched.
var ownerName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

var attemptFileName = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{9}Z-[0-9a-f]{32}\.json$`)
var observationTempName = regexp.MustCompile(`^\.observation-[0-9]+$`)

func pruneRootAttempts(dir string, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if observationTempName.MatchString(entry.Name()) {
			info, err := entry.Info()
			if err == nil && info.Mode().IsRegular() && info.Mode().Perm() == 0600 && now.Sub(info.ModTime()) > rootAttemptRetention {
				_ = os.Remove(filepath.Join(dir, entry.Name()))
			}
			continue
		}
		if entry.Type() != 0 || !strings.HasSuffix(entry.Name(), ".json") || !attemptFileName.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.Mode().IsRegular() && info.Size() <= 65536 && now.Sub(info.ModTime()) > rootAttemptRetention {
			path := filepath.Join(dir, entry.Name())
			b, readErr := os.ReadFile(path)
			var record rootAttemptRecord
			if readErr == nil && len(b) <= 65536 && json.Unmarshal(b, &record) == nil && record.Schema == rootAttemptSchema && record.ID+".json" == entry.Name() && (record.Kind == "install" || record.Kind == "rebaseline" || record.Kind == "activation_observation" || record.Kind == "bundle_publication") {
				_ = os.Remove(path)
			}
		}
	}
}

// Attribute compares verified manifests bound to the identities already
// observed by the caller. Missing/legacy manifests and mismatched identities
// are explicit; presentation text is never parsed as structured evidence.
func (a *Attempt) Attribute(ctx context.Context, serving, candidate string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	p, err := coreactivation.ProbeStateManifest(ctx, serving)
	if err != nil {
		a.record.Attribution = "serving_unavailable"
		return
	}
	c, err := coreactivation.ProbeStateManifest(ctx, candidate)
	if err != nil {
		a.record.Attribution = "candidate_unavailable"
		return
	}
	a.attributeManifests(p, c)
}
func (a *Attempt) attributeManifests(p, c statecontract.Manifest) {
	a.record.Owners = nil
	if p.Contract != a.record.ServingContract || c.Contract != a.record.CandidateContract || p.Contract == "" || c.Contract == "" {
		a.record.Attribution = "identity_mismatch"
		return
	}
	if p.Version != c.Version {
		a.record.Attribution = "format_changed"
		return
	}
	a.record.Attribution = "compared"
	prior := make(map[string]string, len(p.Owners))
	next := make(map[string]string, len(c.Owners))
	for _, o := range p.Owners {
		if !ownerName.MatchString(o.Owner) || len(prior) >= 256 {
			a.record.Attribution = "invalid_or_excess_owners"
			return
		}
		prior[o.Owner] = o.Digest
	}
	for _, o := range c.Owners {
		if !ownerName.MatchString(o.Owner) || len(next) >= 256 {
			a.record.Attribution = "invalid_or_excess_owners"
			return
		}
		next[o.Owner] = o.Digest
	}
	for name, digest := range prior {
		if next[name] != digest {
			a.record.Owners = append(a.record.Owners, name)
		}
	}
	for name := range next {
		if _, ok := prior[name]; !ok {
			a.record.Owners = append(a.record.Owners, name)
		}
	}
	sort.Strings(a.record.Owners)
	if p.Toolchain != c.Toolchain {
		a.record.Attribution = "compared_toolchain_changed"
	}
}
