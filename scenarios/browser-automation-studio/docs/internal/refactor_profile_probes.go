// Isolated profile/replay investigation. Uses real owners with in-memory fake
// filesystem, synthetic driver, and httptest requests; no browser/API is started.
// From api/: GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_profile_probes.go
// Process success means diagnostics completed. Read expected_behavior_met.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/handlers"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"google.golang.org/protobuf/encoding/protojson"
)

type observation struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   any    `json:"actual"`
	Met      bool   `json:"expected_behavior_met"`
}

var observations []observation

func add(id, expected string, actual any, met bool) {
	observations = append(observations, observation{id, expected, actual, met})
}

var (
	oldState = json.RawMessage(`{"cookies":[],"origins":[{"origin":"https://fixture.invalid","localStorage":[{"name":"state","value":"old"}]}]}`)
	newState = json.RawMessage(`{"cookies":[],"origins":[{"origin":"https://fixture.invalid","localStorage":[{"name":"state","value":"new"}]}]}`)
)

const (
	profileRoot = "/synthetic/bas-profile-investigation"
	profileID   = persistence.ProfileID("synthetic-profile")
)

// All credentials are synthetic and process-local, including provider probes.
const syntheticRing = `{"active":1,"keys":{"1":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}`

type probeStore struct{ value string }

func (s *probeStore) Get(string, string) (string, error) { return s.value, nil }
func (s *probeStore) Put(_, _, value string) error       { s.value = value; return nil }
func (s *probeStore) Delete(string, string) error        { s.value = ""; return nil }

var credentials = &probeStore{value: syntheticRing}

func probeAuthority() (*credentialauthority.Authority, error) {
	return credentialauthority.NewAuthority(credentials)
}

type faultFS struct {
	*persistence.MockFileSystem
	truncateProtected bool
}

func (f *faultFS) WriteFileAtomic(name string, data []byte, mode fs.FileMode) error {
	if f.truncateProtected {
		// Fault the atomic writer's staging operation, before publication.
		// api-core/storage separately exercises its real write/sync/rename seam.
		f.SetFile(name+".tmp", data[:5])
		return errors.New("synthetic interrupted profile staging write")
	}
	return f.MockFileSystem.WriteFileAtomic(name, data, mode)
}

func fixture(log *logrus.Logger) (*persistence.FileRepository, *faultFS) {
	credentials.value = syntheticRing
	f := &faultFS{MockFileSystem: persistence.NewMockFileSystem()}
	r := persistence.NewFileRepositoryWithConfig(profileRoot, log, persistence.FileRepositoryConfig{FileSystem: f, Authority: probeAuthority})
	if err := r.Create(&persistence.SessionProfile{ID: profileID, Name: "old-name", StorageState: oldState, CreatedAt: time.Unix(100, 0)}); err != nil {
		panic(err)
	}
	return r, f
}

func freshReader(f *faultFS, log *logrus.Logger) *persistence.FileRepository {
	return persistence.NewFileRepositoryWithConfig(profileRoot, log, persistence.FileRepositoryConfig{FileSystem: f, Authority: probeAuthority})
}

type gatedRepo struct {
	persistence.Repository
	ready, release, competing chan struct{}
	admitted                  atomic.Int32
}

func (g *gatedRepo) Update(id persistence.ProfileID, modify func(*persistence.SessionProfile) error) (*persistence.SessionProfile, error) {
	n := g.admitted.Add(1)
	if n == 2 {
		close(g.competing)
	}
	return g.Repository.Update(id, func(p *persistence.SessionProfile) error {
		if err := modify(p); err != nil {
			return err
		}
		if n == 1 {
			close(g.ready)
			<-g.release
		}
		return nil
	})
}

type fakeDriver struct {
	handlers.RecordModeService
	closed bool
}

func (*fakeDriver) GetStorageState(context.Context, string) (json.RawMessage, error) {
	return newState, nil
}

func (*fakeDriver) GetOpenPages(string) ([]*domain.Page, uuid.UUID, error) {
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	return []*domain.Page{{ID: id, URL: "https://fixture.invalid/tab", Title: "Synthetic tab"}}, id, nil
}
func (d *fakeDriver) CloseSession(context.Context, string) error { d.closed = true; return nil }
func request() *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/synthetic/persist", nil)
	c := chi.NewRouteContext()
	c.URLParams.Add("sessionId", "synthetic-session")
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, c))
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func profileProbes(log *logrus.Logger) {
	r, f := fixture(log)
	p, err := freshReader(f, log).Get(profileID)
	metadata, _ := f.ReadFile(filepath.Join(profileRoot, string(profileID)+".json"))
	var document struct {
		Sealed []byte `json:"sealed"`
	}
	_ = json.Unmarshal(metadata, &document)
	protected := document.Sealed
	add("profile-encryption-roundtrip-control", "Fresh repository restores synthetic state without putting it in plaintext files", map[string]any{"read_error": errorText(err), "state_restored": bytes.Equal(p.StorageState, oldState), "metadata_contains_state": bytes.Contains(metadata, []byte("fixture.invalid")), "protected_contains_state": bytes.Contains(protected, []byte("fixture.invalid"))}, err == nil && bytes.Equal(p.StorageState, oldState) && !bytes.Contains(metadata, []byte("fixture.invalid")) && !bytes.Contains(protected, []byte("fixture.invalid")))
	p.Name = "new-name"
	p.StorageState = newState
	f.RenameErr = errors.New("synthetic metadata commit failure")
	_, saveErr := r.Update(profileID, func(current *persistence.SessionProfile) error { *current = *p; return nil })
	after, readErr := freshReader(f, log).Get(profileID)
	add("profile-metadata-commit-failure", "Failed save preserves one coherent previous profile", map[string]any{"save_error": errorText(saveErr), "read_error": errorText(readErr), "metadata_name": after.Name, "state_is_new": bytes.Equal(after.StorageState, newState)}, saveErr != nil && readErr == nil && after.Name == "old-name" && bytes.Equal(after.StorageState, oldState))

	r, f = fixture(log)
	p, _ = r.Get(profileID)
	p.StorageState = newState
	f.truncateProtected = true
	_, saveErr = r.Update(profileID, func(current *persistence.SessionProfile) error { *current = *p; return nil })
	after, readErr = freshReader(f, log).Get(profileID)
	priorStatePreserved := after != nil && bytes.Equal(after.StorageState, oldState)
	add("profile-interrupted-state-write", "Interrupted state save leaves previous acknowledged profile readable", map[string]any{"save_error": errorText(saveErr), "subsequent_read_error": errorText(readErr), "prior_state_preserved": priorStatePreserved}, saveErr != nil && readErr == nil && priorStatePreserved)

	r, f = fixture(log)
	f.SetFile(filepath.Join(profileRoot, string(profileID)+".json"), []byte(`{"version":1}`))
	after, readErr = r.Get(profileID)
	storageBytes := 0
	if after != nil {
		storageBytes = len(after.StorageState)
	}
	add("profile-missing-protected-state", "A profile whose acknowledged protected state is missing reports incomplete recovery", map[string]any{"read_error": errorText(readErr), "profile_returned": after != nil, "restored_storage_bytes": storageBytes}, readErr != nil)

	r, _ = fixture(log)
	credentials.value = `{"active":1,"keys":{"1":"AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="}}`
	_, readErr = r.Get(profileID)
	listed, listErr := r.List()
	replacement, resolutionErr := sessionprofile.NewService(r, log).GetOrCreateProfile("")
	add("profile-unreadable-list", "Listing exposes recovery failure rather than an apparently empty profile store", map[string]any{"direct_read_error": errorText(readErr), "list_error": errorText(listErr), "listed_count": len(listed), "default_resolution_error": errorText(resolutionErr), "default_created_replacement": replacement != nil && replacement.ID != profileID}, listErr != nil)
	credentials.value = syntheticRing

	r, _ = fixture(log)
	g := &gatedRepo{Repository: r, ready: make(chan struct{}), release: make(chan struct{}), competing: make(chan struct{})}
	svc := sessionprofile.NewService(g, log)
	done := make(chan error, 1)
	go func() {
		_, err := svc.SaveOpenTabs(profileID, []persistence.TabState{{URL: "https://fixture.invalid/new-tab"}})
		done <- err
	}()
	select {
	case <-g.ready:
	case <-time.After(time.Second):
		panic("tab save did not reach gate")
	}
	storageDone := make(chan error, 1)
	go func() { _, err := svc.SaveStorageState(profileID, newState); storageDone <- err }()
	select {
	case <-g.competing:
	case <-time.After(time.Second):
		panic("storage update was not admitted")
	}
	close(g.release)
	storageErr := <-storageDone
	tabsErr := <-done
	after, _ = r.Get(profileID)
	add("profile-concurrent-field-updates", "Successful tab and storage updates both survive a legal read-modify-save interleaving", map[string]any{"storage_error": errorText(storageErr), "tabs_error": errorText(tabsErr), "storage_reverted_to_old": bytes.Equal(after.StorageState, oldState), "new_tab_retained": len(after.OpenTabs) == 1}, storageErr == nil && tabsErr == nil && bytes.Equal(after.StorageState, newState) && len(after.OpenTabs) == 1)

	// Actual HTTP handlers, fake driver and memory-only repository with failed saves.
	memory := persistence.NewMockRepository()
	_ = memory.Create(&persistence.SessionProfile{ID: profileID, StorageState: oldState})
	profileService := sessionprofile.NewService(memory, log)
	profileService.SetActiveSession("synthetic-session", string(profileID))
	driver := &fakeDriver{}
	h := handlers.NewHandlerWithDeps(nil, nil, log, nil, handlers.HandlerDeps{RecordModeService: driver, SessionProfileService: profileService})
	memory.SaveErr = errors.New("synthetic profile save failure")
	response := httptest.NewRecorder()
	h.PersistRecordingSession(response, request())
	add("profile-persist-http-receipt", "Persist endpoint reports failed durable save", map[string]any{"http_status": response.Code, "response": strings.TrimSpace(response.Body.String())}, response.Code >= 400)
	response = httptest.NewRecorder()
	h.CloseRecordingSession(response, request())
	add("profile-close-save-failure", "Failed final persistence remains visible and recoverable when closing", map[string]any{"http_status": response.Code, "driver_closed": driver.closed, "profile_association_after_close": profileService.GetActiveSession("synthetic-session"), "response": strings.TrimSpace(response.Body.String())}, response.Code >= 400 && profileService.GetActiveSession("synthetic-session") != "")

	blank := persistence.NewFileRepositoryWithConfig(profileRoot, log, persistence.FileRepositoryConfig{FileSystem: persistence.NewMockFileSystem(), Authority: func() (*credentialauthority.Authority, error) {
		return credentialauthority.Unavailable("synthetic absent provider")
	}})
	keyErr := blank.Create(&persistence.SessionProfile{ID: "missing-key-control"})
	add("profile-missing-key-fail-closed-control", "Profile creation requires an encryption key", errorText(keyErr), keyErr != nil)
	credentials.value = syntheticRing
}

func replayProbes() {
	type sample struct {
		id, kind string
		payload  map[string]any
		check    func(map[string]any, error) bool
		expected string
	}
	param := func(a map[string]any, k string) map[string]any { v, _ := a[k].(map[string]any); return v }
	samples := []sample{
		{"replay-simple-click-control", "click", map[string]any{"button": "left"}, func(a map[string]any, e error) bool { return e == nil && param(a, "click")["selector"] == "#fixture" }, "Basic click remains a valid typed click"},
		{"replay-click-modifiers", "click", map[string]any{"button": "left", "modifiers": []string{"ctrl"}}, func(a map[string]any, e error) bool { return e == nil && param(a, "click")["modifiers"] != nil }, "Recorded Control-click preserves its modifier"},
		{"replay-double-click", "click", map[string]any{"button": "left", "clickCount": 2}, func(a map[string]any, e error) bool {
			return e == nil && param(a, "click")["click_count"] == float64(2)
		}, "Recorded double-click preserves click count two"},
		{"replay-keyboard-modifiers", "keyboard", map[string]any{"key": "a", "modifiers": []string{"ctrl"}}, func(a map[string]any, e error) bool { return e == nil && param(a, "keyboard")["modifiers"] != nil }, "Recorded Control-A preserves its modifier"},
		{"replay-horizontal-scroll", "scroll", map[string]any{"scrollX": float64(200), "scrollY": float64(400)}, func(a map[string]any, e error) bool { return e == nil && param(a, "scroll")["x"] == float64(200) }, "Recorded scroll preserves both axes"},
		{"replay-blur", "blur", nil, func(a map[string]any, e error) bool { return e != nil || a["type"] == "ACTION_TYPE_BLUR" }, "Blur becomes blur or explicit unsupported failure, never click"},
		{"replay-browser-drag-event", "drag-drop", map[string]any{"phase": "drop", "sourceSelector": "#source", "targetSelector": "#target"}, func(a map[string]any, e error) bool { return e != nil || a["type"] == "ACTION_TYPE_DRAG_DROP" }, "Browser-script drag-drop becomes drag/drop or explicit unsupported failure"},
		{"replay-registry-drag-event", "dragDrop", map[string]any{"targetSelector": "#target"}, func(a map[string]any, e error) bool { return e == nil && a["type"] == "ACTION_TYPE_DRAG_DROP" }, "Registered dragDrop conversion produces valid typed workflow"},
	}
	g := livecapture.NewWorkflowGenerator()
	for _, s := range samples {
		flow, err := g.GenerateWorkflow([]driver.RecordedAction{{ActionType: s.kind, Selector: &driver.SelectorSet{Primary: "#fixture"}, Payload: s.payload}})
		a := map[string]any{}
		if err == nil {
			encoded, _ := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(flow)
			var ingress map[string]any
			_ = json.Unmarshal(encoded, &ingress)
			_, err = workflow.BuildFlowDefinitionV2ForWrite(ingress, nil, nil)
			a = ingress["nodes"].([]any)[0].(map[string]any)["action"].(map[string]any)
		}
		add(s.id, s.expected, map[string]any{"generated_action": a, "typed_ingress_error": errorText(err)}, s.check(a, err))
	}
	flow, err := g.GenerateWorkflow([]driver.RecordedAction{
		{ActionType: "click", PageID: "page-one", FrameID: "frame-one", Selector: &driver.SelectorSet{Primary: "#one"}},
		{ActionType: "click", PageID: "page-two", FrameID: "frame-two", Selector: &driver.SelectorSet{Primary: "#two"}},
	})
	encoded, _ := json.Marshal(flow)
	add("replay-target-context", "Target context is retained or its loss is rejected instead of accepting an untargeted flow", map[string]any{"flow": flow, "typed_ingress_error": errorText(err)}, err != nil || bytes.Contains(encoded, []byte("page-two")) || bytes.Contains(encoded, []byte("frame-two")))
}

func main() {
	log := logrus.New()
	log.SetOutput(io.Discard)
	logrus.SetOutput(io.Discard)
	profileProbes(log)
	replayProbes()
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"schema_version": 1, "observed_at": time.Now().UTC(), "scope": "isolated actual profile repository/service/HTTP handlers and workflow generator/typed ingress; synthetic I/O only", "results": observations})
}
