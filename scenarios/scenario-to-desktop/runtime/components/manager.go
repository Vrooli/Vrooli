// Package components owns optional code inside one desktop installation. It
// admits signed immutable generations, then merges them only at the next boot.
package components

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

const (
	Available   = "available"
	Installing  = "installing"
	Pending     = "pending_next_start"
	Installed   = "installed"
	NeedsUpdate = "needs_update"
	Removing    = "removing"
	OutOfRange  = "out_of_range"
)

type Installation struct {
	Signed             component.SignedEntry `json:"signed"`
	Generation         string                `json:"generation"`
	State              string                `json:"state"`
	Reason             string                `json:"reason,omitempty"`
	Previous           *Generation           `json:"previous,omitempty"`
	ActivationPending  bool                  `json:"activation_pending,omitempty"`
	ActivationDisabled bool                  `json:"activation_disabled,omitempty"`
}

type Generation struct {
	Signed     component.SignedEntry `json:"signed"`
	Generation string                `json:"generation"`
}

var ErrActivationSuperseded = errors.New("component activation superseded by a staged generation")

type Manager struct {
	Root    string
	Version string
	Source  *component.Source
	Client  *http.Client
}

func New(appData, version string, source *component.Source) *Manager {
	return &Manager{Root: filepath.Join(appData, "components"), Version: version, Source: source, Client: &http.Client{Timeout: 3 * time.Minute}}
}

func (m *Manager) lock(ctx context.Context) (func(), error) {
	return artifactlock.AcquireWithPathContext(ctx, m.Root, func(string) (string, error) { return filepath.Join(m.Root, "installation.lock"), nil })
}

func (m *Manager) read() (map[string]Installation, error) {
	result := map[string]Installation{}
	b, err := os.ReadFile(filepath.Join(m.Root, "installations.json"))
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("component installation state: %w", err)
	}
	for id, row := range result {
		if !component.ValidID(id) || row.Signed.Entry.ID != id || row.Generation != row.Signed.Entry.SHA256 {
			return nil, fmt.Errorf("invalid component installation state")
		}
	}
	return result, nil
}

func (m *Manager) save(rows map[string]Installation) error {
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(m.Root, 0o700); err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(filepath.Join(m.Root, "installations.json"), b, 0o600)
}

func (m *Manager) Catalog(ctx context.Context) ([]component.SignedEntry, error) {
	if m.Source == nil || m.Source.CatalogURL == "" {
		return []component.SignedEntry{}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.Source.CatalogURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("component catalog unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("component catalog HTTP %d", resp.StatusCode)
	}
	var rows []component.SignedEntry
	if err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&rows); err != nil {
		return nil, fmt.Errorf("invalid component catalog")
	}
	seen := map[string]bool{}
	filtered := make([]component.SignedEntry, 0, len(rows))
	for _, row := range rows {
		if err = row.Verify(m.Source.PublicKeyPEM); err != nil {
			return nil, fmt.Errorf("component catalog signature: %w", err)
		}
		if row.Entry.OS != runtime.GOOS || row.Entry.Arch != runtime.GOARCH {
			continue
		}
		if seen[row.Entry.ID] {
			return nil, fmt.Errorf("ambiguous component release %s", row.Entry.ID)
		}
		seen[row.Entry.ID] = true
		filtered = append(filtered, row)
	}
	return filtered, nil
}

// Install verifies the catalog entry before any network or file mutation. The
// existing pointer is changed only after both archive and extracted tree verify.
func (m *Manager) Install(ctx context.Context, signed component.SignedEntry) (Installation, error) {
	if m.Source == nil {
		return Installation{}, fmt.Errorf("component source is not configured")
	}
	if err := signed.Verify(m.Source.PublicKeyPEM); err != nil {
		return Installation{}, err
	}
	e := signed.Entry
	if !e.Compatible(m.Version, runtime.GOOS, runtime.GOARCH) {
		return Installation{}, fmt.Errorf("component is outside this host's supported range or platform")
	}
	var bundle manifest.Manifest
	if err := json.Unmarshal(e.Manifest, &bundle); err != nil {
		return Installation{}, err
	}
	if bundle.App.Scenario != e.Owner || bundle.App.Version != e.Version {
		return Installation{}, fmt.Errorf("component owner/version does not match its service manifest")
	}
	if err := bundle.Validate(runtime.GOOS, runtime.GOARCH); err != nil {
		return Installation{}, err
	}
	release, err := m.lock(ctx)
	if err != nil {
		return Installation{}, err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return Installation{}, err
	}
	stage, err := os.MkdirTemp(m.Root, ".install-")
	if err != nil {
		return Installation{}, err
	}
	defer os.RemoveAll(stage)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.ArtifactURL, nil)
	if err != nil {
		return Installation{}, err
	}
	response, err := m.Client.Do(req)
	if err != nil {
		return Installation{}, fmt.Errorf("component download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Installation{}, fmt.Errorf("component artifact HTTP %d", response.StatusCode)
	}
	archive := filepath.Join(stage, "component.tar.gz")
	file, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return Installation{}, err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, e.Size+1))
	closeErr := file.Close()
	if copyErr != nil {
		return Installation{}, copyErr
	}
	if closeErr != nil {
		return Installation{}, closeErr
	}
	if size != e.Size || hex.EncodeToString(hash.Sum(nil)) != e.SHA256 {
		return Installation{}, fmt.Errorf("component archive checksum or size mismatch")
	}
	tree := filepath.Join(stage, "bundle")
	limits := binaryfetch.ExtractOptions{MaxEntries: 50000, MaxExpandedBytes: 2 << 30, MaxEntryBytes: 512 << 20, Deadline: time.Now().Add(2 * time.Minute)}
	summary, err := binaryfetch.InspectArchiveBounded(archive, "tar.gz", limits)
	if err != nil {
		return Installation{}, err
	}
	if summary.Symlinks != 0 || summary.Hardlinks != 0 {
		return Installation{}, fmt.Errorf("component archives must contain regular files and directories only")
	}
	if _, err = binaryfetch.ExtractArchiveBounded(archive, "tar.gz", tree, limits); err != nil {
		return Installation{}, err
	}
	got, err := binaryfetch.TreeDigest(tree)
	if err != nil {
		return Installation{}, err
	}
	if got != e.TreeSHA256 {
		return Installation{}, fmt.Errorf("component extracted tree digest mismatch")
	}
	raw, err := os.ReadFile(filepath.Join(tree, "bundle.json"))
	if err != nil {
		return Installation{}, err
	}
	var actual, expected any
	if json.Unmarshal(raw, &actual) != nil || json.Unmarshal(e.Manifest, &expected) != nil {
		return Installation{}, fmt.Errorf("invalid component manifest")
	}
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(expected)
	if !bytes.Equal(a, b) {
		return Installation{}, fmt.Errorf("component service manifest differs from the signed entry")
	}
	generation := filepath.Join(m.Root, e.ID, "generations", e.SHA256)
	if err = os.MkdirAll(filepath.Dir(generation), 0o700); err != nil {
		return Installation{}, err
	}
	if _, err = os.Stat(generation); os.IsNotExist(err) {
		if err = os.Rename(tree, generation); err != nil {
			return Installation{}, err
		}
	} else if err != nil {
		return Installation{}, err
	} else if got, err = binaryfetch.TreeDigest(generation); err != nil || got != e.TreeSHA256 {
		return Installation{}, fmt.Errorf("existing component generation failed verification")
	}
	result := Installation{Signed: signed, Generation: e.SHA256, State: Pending}
	if old, ok := rows[e.ID]; ok {
		if old.Previous != nil {
			// Retrying the same staged generation must retain the last healthy
			// fallback until readiness commits that generation.
			result.Previous = old.Previous
		} else if old.Generation != result.Generation && old.State == Installed {
			result.Previous = &Generation{Signed: old.Signed, Generation: old.Generation}
		}
	}
	rows[e.ID] = result
	if err = m.save(rows); err != nil {
		return Installation{}, err
	}
	return result, nil
}

// CompleteActivation forgets the fallback only after the runtime has observed
// the actual new service graph become ready.
func (m *Manager) CompleteActivation(ctx context.Context, id, generation string) error {
	release, err := m.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return err
	}
	row, ok := rows[id]
	if !ok {
		return fmt.Errorf("component changed during activation")
	}
	if row.Generation != generation {
		return ErrActivationSuperseded
	}
	row.Previous = nil
	row.ActivationPending = false
	rows[id] = row
	return m.save(rows)
}

// Disable leaves an unavailable optional generation off until an explicit
// install/update retries it. Its user data and verified code remain removable.
func (m *Manager) Disable(ctx context.Context, id, generation, reason string) error {
	release, err := m.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return err
	}
	row, ok := rows[id]
	if !ok {
		return fmt.Errorf("component removed during activation")
	}
	if row.Generation != generation {
		return ErrActivationSuperseded
	}
	row.State, row.Reason, row.Previous = NeedsUpdate, reason, nil
	row.ActivationPending, row.ActivationDisabled = false, true
	rows[id] = row
	return m.save(rows)
}

// Rollback re-admits the retained signed generation before selecting it again.
func (m *Manager) Rollback(ctx context.Context, id, generation string) (Installation, error) {
	release, err := m.lock(ctx)
	if err != nil {
		return Installation{}, err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return Installation{}, err
	}
	row, ok := rows[id]
	if !ok || row.Generation != generation || row.Previous == nil {
		return Installation{}, fmt.Errorf("no retained component generation")
	}
	old := row.Previous
	if old.Signed.Entry.ID != id || old.Generation != old.Signed.Entry.SHA256 {
		return Installation{}, fmt.Errorf("retained component identity differs")
	}
	if m.Source == nil || old.Signed.Verify(m.Source.PublicKeyPEM) != nil || !old.Signed.Entry.Compatible(m.Version, runtime.GOOS, runtime.GOARCH) {
		return Installation{}, fmt.Errorf("retained component generation is not trusted or compatible")
	}
	root := filepath.Join(m.Root, id, "generations", old.Generation)
	if got, err := binaryfetch.TreeDigest(root); err != nil || got != old.Signed.Entry.TreeSHA256 {
		return Installation{}, fmt.Errorf("retained component generation failed verification")
	}
	row = Installation{Signed: old.Signed, Generation: old.Generation, State: Installed, Reason: "The update could not start; the previous version was restored."}
	rows[id] = row
	if err := m.save(rows); err != nil {
		return Installation{}, err
	}
	return row, nil
}

func (m *Manager) Installed(ctx context.Context) ([]Installation, error) {
	release, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return nil, err
	}
	result := make([]Installation, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Signed.Entry.ID < result[j].Signed.Entry.ID })
	return result, nil
}

// Merge checks stored signatures and trees on every boot. Bad/incompatible
// optional code stays off while the host remains usable.
func (m *Manager) Merge(ctx context.Context, host *manifest.Manifest) error {
	if m.Source == nil {
		return nil
	}
	release, err := m.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(rows))
	for id := range rows {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	seen := map[string]bool{}
	for _, svc := range host.Services {
		seen[svc.ID] = true
	}
	for _, id := range ids {
		row := rows[id]
		e := row.Signed.Entry
		if row.ActivationDisabled {
			continue
		}
		if row.State == Pending {
			row.ActivationPending = true
		}
		row.Reason = ""
		root := filepath.Join(m.Root, id, "generations", row.Generation)
		if err = row.Signed.Verify(m.Source.PublicKeyPEM); err != nil {
			row.State = NeedsUpdate
			row.Reason = "stored signature failed verification"
			rows[id] = row
			continue
		}
		if !e.Compatible(m.Version, runtime.GOOS, runtime.GOARCH) {
			row.State = OutOfRange
			row.Reason = "component does not support this host version"
			rows[id] = row
			continue
		}
		if got, hashErr := binaryfetch.TreeDigest(root); hashErr != nil || got != e.TreeSHA256 {
			row.State = NeedsUpdate
			row.Reason = "installed tree failed verification"
			rows[id] = row
			continue
		}
		var bundle manifest.Manifest
		if err = json.Unmarshal(e.Manifest, &bundle); err != nil {
			return err
		}
		projected, err := manifest.ProjectComponent(e.Owner, bundle.Services)
		if err != nil {
			return err
		}
		collision := false
		for _, svc := range projected {
			if seen[svc.ID] {
				collision = true
			}
		}
		if collision {
			row.State = NeedsUpdate
			row.Reason = "component service collides with a bundled service"
			rows[id] = row
			continue
		}
		for i := range projected {
			projected[i].BundleRoot = root
			projected[i].ComponentID = id
			projected[i].ComponentVersion = e.Version
			projected[i].ComponentSlice = e.Slice
			seen[projected[i].ID] = true
		}
		host.Services = append(host.Services, projected...)
		row.State = Installed
		rows[id] = row
	}
	return m.save(rows)
}

// Remove is called only after the supervisor has stopped this component's
// services. Unpublishing the pointer precedes code deletion; data is separate.
func (m *Manager) Remove(ctx context.Context, id string, keepData bool) error {
	if !component.ValidID(id) {
		return fmt.Errorf("invalid component ID")
	}
	release, err := m.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	rows, err := m.read()
	if err != nil {
		return err
	}
	delete(rows, id)
	if err = m.save(rows); err != nil {
		return err
	}
	if err = os.RemoveAll(filepath.Join(m.Root, id, "generations")); err != nil {
		return err
	}
	if !keepData {
		return os.RemoveAll(filepath.Join(m.Root, id, "data"))
	}
	return nil
}
