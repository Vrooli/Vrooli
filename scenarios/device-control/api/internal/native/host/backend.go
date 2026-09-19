// Package host provides the capture and input boundary for native user
// sessions that do not expose the Linux X11 wire protocol.  Commands are
// invoked directly with bounded contexts; no shell is involved.  The helper
// remains the authenticated owner of this backend, so callers cannot select a
// different process or desktop from an action payload.
package host

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"device-control/internal/sessions"
	"github.com/google/uuid"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	maxCaptureBytes  = 32 << 20
	maxCapturePixels = 64 << 20
	commandTimeout   = 5 * time.Second
)

var (
	ErrUnavailable      = errors.New("native host desktop unavailable")
	ErrPermissionDenied = errors.New("native host desktop permission denied")
)

type (
	commandFn func(context.Context, string, ...string) ([]byte, error)
	runFn     func(context.Context, string, ...string) error
	captureFn func(context.Context) ([]byte, error)
)

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > maxCaptureBytes-b.Len() {
		return 0, ErrUnavailable
	}
	return b.Buffer.Write(p)
}

// Backend is deliberately limited to the current process's logged-in user
// session.  It does not discover or switch sessions and it never treats a
// successful executable lookup as permission evidence.
type Backend struct {
	mu                                           sync.Mutex
	platform                                     string
	command                                      commandFn
	run                                          runFn
	nativeCapture                                captureFn
	display                                      string
	session                                      string
	revision                                     string
	width                                        int
	height                                       int
	apps                                         map[string]windowsApplication
	appRev, appLease                             string
	appExpires                                   time.Time
	elements                                     map[string]windowsElement
	elementOrder                                 []string
	semanticRev, semanticLease, semanticGeometry string
	semanticProcessID                            uint32
	semanticExpires                              time.Time
	activationRootX, activationRootY             int32
	heldKeys                                     map[string]uint16
	heldButtons                                  map[desktopv1.PointerAction_Button]bool
	closeOnce                                    sync.Once
	closeErr                                     error
}

type windowsApplication struct {
	name, runtimeID string
	processID       uint32
}

type windowsElement struct {
	runtimeID string
	name      string
	text      string
	editable  bool
	role      uint32
	parentID  string
	windowID  string
}

type windowsApplicationRecord struct {
	Name      string `json:"name"`
	RuntimeID string `json:"runtime_id"`
	ProcessID uint32 `json:"process_id"`
}

type windowsElementRecord struct {
	Name        string `json:"name"`
	RuntimeID   string `json:"runtime_id"`
	Value       string `json:"value"`
	Editable    bool   `json:"editable"`
	Role        uint32 `json:"role"`
	ParentIndex int    `json:"parent_index"`
	WindowIndex int    `json:"window_index"`
}

// activationRecord is deliberately a narrow wire shape emitted by the native
// user-session probe. Window and process identities stay inside this package;
// the sessions layer receives only the short-lived evidence below.
type activationRecord struct {
	ActiveWindow  uint64 `json:"active_window"`
	ProcessID     uint32 `json:"process_id"`
	PointerWindow uint64 `json:"pointer_window"`
	PointerX      int32  `json:"pointer_x"`
	PointerY      int32  `json:"pointer_y"`
	X             int32  `json:"x"`
	Y             int32  `json:"y"`
	Width         uint32 `json:"width"`
	Height        uint32 `json:"height"`
	RootX         int32  `json:"root_x"`
	RootY         int32  `json:"root_y"`
}

func New() (*Backend, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return nil, ErrUnavailable
	}
	return NewForSession("")
}

// NewForSession binds the backend to the session identifier admitted by the
// control plane. An empty identifier is accepted only for isolated tests; the
// lifecycle helper always supplies the configured session.
func NewForSession(sessionID string) (*Backend, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return nil, ErrUnavailable
	}
	b := newBackendForPlatform(runtime.GOOS, runCommand, runCommandError)
	if runtime.GOOS == "darwin" {
		b.nativeCapture = screenCaptureKitCapture
	}
	b.session = strings.TrimSpace(sessionID)
	return b, nil
}

func newBackendForPlatform(platform string, command commandFn, run runFn) *Backend {
	return &Backend{platform: platform, command: command, run: run, display: platform + "-current-user-display", heldKeys: make(map[string]uint16), heldButtons: make(map[desktopv1.PointerAction_Button]bool)}
}

// CheckSession is used by the desktop lifecycle monitor. It checks that the
// helper still belongs to an interactive user session and, when configured,
// that the OS reports the same session identifier. It never switches users or
// repairs a locked desktop.
func (b *Backend) CheckSession(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	if b.platform == "darwin" {
		data, err := b.command(ctx, "stat", "-f", "%u", "/dev/console")
		if err != nil || strings.TrimSpace(string(data)) != strconv.Itoa(os.Getuid()) {
			return ErrUnavailable
		}
		return nil
	}
	data, err := b.command(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[System.Diagnostics.Process]::GetCurrentProcess().SessionId")
	if err != nil || strings.TrimSpace(string(data)) == "" || strings.TrimSpace(string(data)) == "0" {
		return ErrUnavailable
	}
	if b.session != "" && strings.TrimSpace(string(data)) != b.session {
		return ErrUnavailable
	}
	return nil
}

func (b *Backend) Observe(ctx context.Context) (sessions.DesktopObservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return sessions.DesktopObservation{}, err
		}
	}
	snapshot, err := b.observePixelsLocked(ctx)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return sessions.DesktopObservation{}, err
		}
	}
	return snapshot, nil
}

func (b *Backend) observePixelsLocked(ctx context.Context) (sessions.DesktopObservation, error) {
	data, err := b.capture(ctx)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	img, err := decodePNG(data)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	b.width, b.height = img.Bounds().Dx(), img.Bounds().Dy()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d", b.display, b.width, b.height)))
	b.revision = fmt.Sprintf("%x", hash[:])
	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return sessions.DesktopObservation{Image: rgba, DisplayID: b.display, GeometryRevision: b.revision, CapturedAt: time.Now().UTC()}, nil
}

// Applications enumerates top-level UI Automation windows in the current
// interactive Windows session. IDs are helper-owned and expire with the
// short-lived catalog; callers never receive a RuntimeId or HWND to reuse.
func (b *Backend) Applications(ctx context.Context, leaseID string) (sessions.DesktopApplications, error) {
	if (b.platform != "windows" && b.platform != "darwin") || strings.TrimSpace(leaseID) == "" {
		return sessions.DesktopApplications{}, ErrUnavailable
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.CheckSession(ctx); err != nil {
		return sessions.DesktopApplications{}, err
	}
	applicationCommand := windowsApplicationsCommand()
	if b.platform == "darwin" {
		applicationCommand = macOSApplicationsCommand()
	}
	data, err := b.commandWithTimeout(ctx, applicationCommand)
	if err != nil {
		return sessions.DesktopApplications{}, err
	}
	records, err := decodeWindowsApplications(data)
	if err != nil || len(records) == 0 || len(records) > 128 {
		return sessions.DesktopApplications{}, ErrUnavailable
	}
	revision := uuid.NewString()
	apps := make(map[string]windowsApplication, len(records))
	result := sessions.DesktopApplications{Revision: revision, ExpiresAt: time.Now().Add(30 * time.Second)}
	for _, record := range records {
		if strings.TrimSpace(record.RuntimeID) == "" || record.ProcessID == 0 || len(record.Name) > 4096 || !b.validRuntimeID(record.RuntimeID) {
			return sessions.DesktopApplications{}, ErrUnavailable
		}
		id := uuid.NewString()
		apps[id] = windowsApplication{name: record.Name, runtimeID: record.RuntimeID, processID: record.ProcessID}
		result.Applications = append(result.Applications, sessions.DesktopApplication{ID: id, Name: record.Name, ProcessID: record.ProcessID})
	}
	if err := b.CheckSession(ctx); err != nil {
		return sessions.DesktopApplications{}, err
	}
	b.apps, b.appRev, b.appLease, b.appExpires = apps, revision, leaseID, result.ExpiresAt
	return result, nil
}

func (b *Backend) ObserveApplication(ctx context.Context, applicationID, revision, leaseID string) (sessions.DesktopObservation, error) {
	if (b.platform != "windows" && b.platform != "darwin") || strings.TrimSpace(leaseID) == "" {
		return sessions.DesktopObservation{}, ErrUnavailable
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	app, ok := b.apps[applicationID]
	if !ok || revision == "" || revision != b.appRev || leaseID != b.appLease || !time.Now().Before(b.appExpires) {
		return sessions.DesktopObservation{}, ErrUnavailable
	}
	if err := b.CheckSession(ctx); err != nil {
		return sessions.DesktopObservation{}, err
	}
	snapshot, err := b.observePixelsLocked(ctx)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	semantic, err := b.observeWindowsSemanticLocked(ctx, app.runtimeID, app.processID, leaseID, snapshot.GeometryRevision)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	if err := b.CheckSession(ctx); err != nil {
		return sessions.DesktopObservation{}, err
	}
	snapshot.Semantic = semantic
	return snapshot, nil
}

func (b *Backend) ObserveProcess(ctx context.Context, processID uint32, leaseID string) (sessions.DesktopObservation, error) {
	if (b.platform != "windows" && b.platform != "darwin") || processID == 0 || strings.TrimSpace(leaseID) == "" {
		return sessions.DesktopObservation{}, ErrUnavailable
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.CheckSession(ctx); err != nil {
		return sessions.DesktopObservation{}, err
	}
	snapshot, err := b.observePixelsLocked(ctx)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	processCommand := windowsApplicationForProcessCommand(processID)
	if b.platform == "darwin" {
		processCommand = macOSApplicationForProcessCommand(processID)
	}
	data, err := b.commandWithTimeout(ctx, processCommand)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	records, err := decodeWindowsApplications(data)
	if err != nil || len(records) != 1 || records[0].ProcessID != processID {
		return sessions.DesktopObservation{}, ErrUnavailable
	}
	semantic, err := b.observeWindowsSemanticLocked(ctx, records[0].RuntimeID, processID, leaseID, snapshot.GeometryRevision)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	snapshot.Semantic = semantic
	return snapshot, nil
}

func (b *Backend) observeWindowsSemanticLocked(ctx context.Context, runtimeID string, processID uint32, leaseID, geometry string) (*sessions.DesktopSemanticObservation, error) {
	if !b.validRuntimeID(runtimeID) {
		return nil, ErrUnavailable
	}
	semanticCommand := windowsSemanticCommand(runtimeID)
	if b.platform == "darwin" {
		semanticCommand = macOSSemanticCommand(runtimeID)
	}
	data, err := b.commandWithTimeout(ctx, semanticCommand)
	if err != nil {
		return nil, err
	}
	records, err := decodeWindowsElements(data)
	if err != nil || len(records) == 0 || len(records) > 128 {
		return nil, ErrUnavailable
	}
	revision := uuid.NewString()
	ids := make([]string, len(records))
	for i, record := range records {
		if !b.validRuntimeID(record.RuntimeID) || len(record.Name) > 4096 || record.ParentIndex < -1 || record.ParentIndex >= i || record.WindowIndex < 0 || record.WindowIndex >= len(records) {
			return nil, ErrUnavailable
		}
		ids[i] = uuid.NewString()
	}
	elements := make(map[string]windowsElement, len(records))
	elementOrder := make([]string, len(records))
	semantic := &sessions.DesktopSemanticObservation{Revision: revision, ProcessID: processID, ExpiresAt: time.Now().Add(5 * time.Second)}
	for i, record := range records {
		parentID := ""
		if record.ParentIndex >= 0 {
			parentID = ids[record.ParentIndex]
		}
		windowID := ids[record.WindowIndex]
		elements[ids[i]] = windowsElement{runtimeID: record.RuntimeID, name: record.Name, text: record.Value, editable: record.Editable, role: roleFromRuntimeRecord(record), parentID: parentID, windowID: windowID}
		elementOrder[i] = ids[i]
		semantic.Elements = append(semantic.Elements, sessions.DesktopSemanticElement{ID: ids[i], Name: record.Name, Editable: record.Editable, ParentID: parentID, WindowID: windowID, Role: roleFromRuntimeRecord(record)})
	}
	b.elements, b.elementOrder, b.semanticRev, b.semanticLease, b.semanticGeometry, b.semanticProcessID, b.semanticExpires = elements, elementOrder, revision, leaseID, geometry, processID, semantic.ExpiresAt
	return semantic, nil
}

// Resolve matches only the helper-owned, unexpired observation. A stale or
// ambiguous selector never falls back to a new tree or a name-only search.
func (b *Backend) Resolve(ctx context.Context, selector sessions.DesktopSelector, leaseID string) (sessions.DesktopResolution, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if (b.platform != "windows" && b.platform != "darwin") || leaseID == "" || leaseID != b.semanticLease || selector.Revision == "" || selector.Revision != b.semanticRev || selector.WindowID == "" || selector.Name == "" || !time.Now().Before(b.semanticExpires) {
		return sessions.DesktopResolution{}, ErrUnavailable
	}
	if err := b.CheckSession(ctx); err != nil {
		return sessions.DesktopResolution{}, err
	}
	result := sessions.DesktopResolution{Disposition: "absent", Revision: b.semanticRev, GeometryRevision: b.semanticGeometry, ExpiresAt: b.semanticExpires}
	for _, id := range b.elementOrder {
		element := b.elements[id]
		if element.windowID == selector.WindowID && element.name == selector.Name && (!selector.EditableOnly || element.editable) {
			result.ElementIDs = append(result.ElementIDs, id)
		}
	}
	switch len(result.ElementIDs) {
	case 1:
		result.Disposition = "unique"
	case 2:
		result.Disposition = "ambiguous"
	default:
		if len(result.ElementIDs) > 2 {
			result.Disposition = "ambiguous"
		}
	}
	return result, nil
}

func (b *Backend) Validate(ctx context.Context, command sessions.DesktopCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return err
		}
	}
	if b.revision == "" || command.GeometryRevision != b.revision {
		return ErrUnavailable
	}
	action, err := decodeAction(command.Payload)
	if err != nil {
		return err
	}
	if action.GetText() != nil || action.GetAssertText() != nil || action.GetInvoke() != nil {
		return b.validateSemanticAction(ctx, command, action)
	}
	return b.validateAction(action)
}

func (b *Backend) Apply(ctx context.Context, command sessions.DesktopCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return err
		}
	}
	if b.revision == "" || command.GeometryRevision != b.revision {
		return ErrUnavailable
	}
	action, err := decodeAction(command.Payload)
	if err != nil {
		return err
	}
	if err := b.validateAction(action); err != nil {
		if action.GetText() == nil && action.GetAssertText() == nil && action.GetInvoke() == nil {
			return err
		}
	}
	if action.GetText() != nil || action.GetAssertText() != nil || action.GetInvoke() != nil {
		if err := b.validateSemanticAction(ctx, command, action); err != nil {
			return err
		}
		if invoke := action.GetInvoke(); invoke != nil {
			element := b.elements[invoke.ElementId]
			if b.session != "" {
				if err := b.CheckSession(ctx); err != nil {
					return err
				}
			}
			if err := b.verifySemanticProcessLocked(ctx); err != nil {
				return err
			}
			if err := b.verifySemanticElementLocked(ctx, invoke.ElementId); err != nil {
				return err
			}
			err := b.invokeSemantic(ctx, element.runtimeID)
			b.elements = nil
			b.elementOrder = nil
			b.semanticRev, b.semanticLease, b.semanticGeometry, b.semanticProcessID = "", "", "", 0
			return err
		}
		if action.GetAssertText() != nil {
			if err := b.verifySemanticProcessLocked(ctx); err != nil {
				return err
			}
			b.elements = nil
			b.elementOrder = nil
			b.semanticRev, b.semanticLease, b.semanticGeometry, b.semanticProcessID = "", "", "", 0
			return nil
		}
		text := action.GetText()
		element := b.elements[text.ElementId]
		if err := b.verifySemanticProcessLocked(ctx); err != nil {
			return err
		}
		if err := b.verifySemanticElementLocked(ctx, text.ElementId); err != nil {
			return err
		}
		current, err := b.readSemanticText(ctx, element.runtimeID)
		if err != nil {
			return err
		}
		if current != element.text {
			return ErrUnavailable
		}
		runes := []rune(current)
		if text.Position < 0 || text.Position > int32(len(runes)) {
			return ErrUnavailable
		}
		updated := string(runes[:text.Position]) + text.Text + string(runes[text.Position:])
		if err := b.verifySemanticProcessLocked(ctx); err != nil {
			return err
		}
		if err := b.verifySemanticElementLocked(ctx, text.ElementId); err != nil {
			return err
		}
		if err := b.setSemanticText(ctx, element.runtimeID, updated); err != nil {
			return err
		}
		b.elements = nil
		b.elementOrder = nil
		b.semanticRev, b.semanticLease, b.semanticGeometry, b.semanticProcessID = "", "", "", 0
		return nil
	}
	if key := action.GetKey(); key != nil {
		return b.applyKey(ctx, key)
	}
	if pointer := action.GetPointer(); pointer != nil && (b.platform == "windows" || b.platform == "darwin") {
		return b.applyPointer(ctx, pointer)
	}
	return b.apply(ctx, action)
}

func (b *Backend) applyPointer(ctx context.Context, pointer *desktopv1.PointerAction) error {
	native := b.applyWindows
	if b.platform == "darwin" {
		native = b.applyMacOS
	}
	if pointer == nil || pointer.Kind == desktopv1.PointerAction_KIND_MOVE {
		return native(ctx, &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: pointer}})
	}
	held := b.heldButtons[pointer.Button]
	if pointer.Kind == desktopv1.PointerAction_KIND_DOWN {
		if held {
			return nil
		}
		b.heldButtons[pointer.Button] = true
		return native(ctx, &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: pointer}})
	}
	if pointer.Kind == desktopv1.PointerAction_KIND_UP {
		if !held {
			return ErrUnavailable
		}
		if err := native(ctx, &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: pointer}}); err != nil {
			return err
		}
		delete(b.heldButtons, pointer.Button)
		return nil
	}
	if held {
		return ErrUnavailable
	}
	b.heldButtons[pointer.Button] = true
	if err := native(ctx, &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: pointer}}); err != nil {
		return err
	}
	delete(b.heldButtons, pointer.Button)
	return nil
}

func (b *Backend) applyKey(ctx context.Context, key *desktopv1.KeyAction) error {
	if key == nil {
		return ErrUnavailable
	}
	if key.Kind == desktopv1.KeyAction_KIND_PRESS {
		return b.applyKeyPress(ctx, key.Key)
	}
	code, ok := keyCode(b.platform, key.Key)
	if !ok {
		return ErrUnavailable
	}
	heldID := heldKeyID(key.Key)
	if key.Kind == desktopv1.KeyAction_KIND_DOWN {
		if b.heldKeys[heldID] != 0 {
			return nil
		}
		// Record before sending; a transport failure may follow the key-down.
		b.heldKeys[heldID] = code
		return b.applyKeyCode(ctx, code, true)
	}
	if key.Kind == desktopv1.KeyAction_KIND_UP {
		if b.heldKeys[heldID] == 0 {
			return ErrUnavailable
		}
		if err := b.applyKeyCode(ctx, code, false); err != nil {
			return err
		}
		delete(b.heldKeys, heldID)
		return nil
	}
	return ErrUnavailable
}

func (b *Backend) applyKeyPress(ctx context.Context, key string) error {
	if b.platform == "darwin" {
		return b.runWithTimeout(ctx, []string{"osascript", "-e", fmt.Sprintf("tell application \"System Events\" to keystroke %s", appleScriptString(key))})
	}
	script := "Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait(" + powershellString(key) + ")"
	return b.runWithTimeout(ctx, []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script})
}

func (b *Backend) applyKeyCode(ctx context.Context, code uint16, down bool) error {
	if b.platform == "darwin" {
		return b.runWithTimeout(ctx, macOSKeyCodeCommand(code, down))
	}
	script := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VrooliKey { [DllImport(\"user32.dll\")] public static extern void keybd_event(byte key, byte scan, uint flags, UIntPtr extra); }'; " + windowsKeyCodeExpression(code, down)
	return b.runWithTimeout(ctx, []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script})
}

func (b *Backend) validateSemanticAction(ctx context.Context, command sessions.DesktopCommand, action *desktopv1.Action) error {
	if (b.platform != "windows" && b.platform != "darwin") || action == nil || command.GeometryRevision == "" || command.GeometryRevision != b.semanticGeometry || command.Lease.Ref.SessionID == "" || command.Lease.Ref.SessionID != b.semanticLease || !time.Now().Before(b.semanticExpires) {
		return ErrUnavailable
	}
	var elementID, observationRevision string
	var expected string
	if text := action.GetText(); text != nil {
		elementID, observationRevision = text.ElementId, text.ObservationRevision
	} else if assertion := action.GetAssertText(); assertion != nil {
		elementID, observationRevision, expected = assertion.ElementId, assertion.ObservationRevision, assertion.ExpectedText
	} else if invoke := action.GetInvoke(); invoke != nil {
		elementID, observationRevision = invoke.ElementId, invoke.ObservationRevision
	} else {
		return ErrUnavailable
	}
	element, ok := b.elements[elementID]
	if !ok || observationRevision != b.semanticRev {
		return ErrUnavailable
	}
	if err := b.verifySemanticProcessLocked(ctx); err != nil {
		return err
	}
	if err := b.verifySemanticElementLocked(ctx, elementID); err != nil {
		return err
	}
	if action.GetInvoke() != nil {
		if !b.validRuntimeID(element.runtimeID) {
			return ErrUnavailable
		}
		return nil
	}
	if !element.editable {
		return ErrUnavailable
	}
	if text := action.GetText(); text != nil {
		if text.Position < 0 {
			return ErrUnavailable
		}
		current, err := b.readSemanticText(ctx, element.runtimeID)
		if err != nil || current != element.text || text.Position > int32(len([]rune(current))) {
			return ErrUnavailable
		}
	}
	if assertion := action.GetAssertText(); assertion != nil {
		current, err := b.readSemanticText(ctx, element.runtimeID)
		if err != nil || current != expected {
			return ErrUnavailable
		}
	}
	return nil
}

func (b *Backend) ReleaseHeld(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var result error
	for key, code := range b.heldKeys {
		if err := b.applyKeyCode(ctx, code, false); err != nil {
			result = errors.Join(result, err)
		} else {
			delete(b.heldKeys, key)
		}
	}
	for button := range b.heldButtons {
		var err error
		if b.platform == "windows" {
			err = b.releaseWindowsButton(ctx, button)
		} else if b.platform == "darwin" {
			err = b.runWithTimeout(ctx, macOSReleaseButtonCommand(button))
		} else {
			delete(b.heldButtons, button)
			continue
		}
		if err != nil {
			result = errors.Join(result, err)
		} else {
			delete(b.heldButtons, button)
		}
	}
	return result
}

func (b *Backend) Close() error {
	b.closeOnce.Do(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		b.closeErr = b.ReleaseHeld(cleanupCtx)
		cancel()
	})
	return b.closeErr
}

// CaptureActivation records the focused application before the companion
// takes focus. It is metadata-only; pixels are captured through the explicit
// image variant so ordinary activation never retains screen bytes.
func (b *Backend) CaptureActivation(ctx context.Context) (sessions.DesktopActivationContext, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.captureActivation(ctx, false)
}

// CaptureActivationImage performs the same bounded metadata probe and captures
// pixels in one helper operation. The sessions layer validates and clones the
// image before it can escape the helper.
func (b *Backend) CaptureActivationImage(ctx context.Context) (sessions.DesktopActivationImage, error) {
	b.mu.Lock()
	metadata, err := b.captureActivation(ctx, true)
	rootX, rootY := b.activationRootX, b.activationRootY
	b.mu.Unlock()
	if err != nil {
		return sessions.DesktopActivationImage{}, err
	}
	data, err := b.capture(ctx)
	if err != nil {
		return sessions.DesktopActivationImage{}, err
	}
	// The screenshot is an independent native operation. Recheck the bound
	// user session before decoding or publishing it so a session transition
	// during capture cannot associate pixels with the prior activation.
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return sessions.DesktopActivationImage{}, err
		}
	}
	decoded, err := decodePNG(data)
	if err != nil {
		return sessions.DesktopActivationImage{}, ErrUnavailable
	}
	originX := int(metadata.SourceBounds.X - rootX)
	originY := int(metadata.SourceBounds.Y - rootY)
	// Native screen captures are rooted at the virtual display origin. The
	// metadata probe uses the same coordinate space; crop only when the exact
	// window rectangle is inside that image. A mismatched frame is refused.
	windowRect := image.Rect(originX, originY, originX+int(metadata.SourceBounds.Width), originY+int(metadata.SourceBounds.Height))
	if !windowRect.In(decoded.Bounds()) {
		return sessions.DesktopActivationImage{}, ErrUnavailable
	}
	rgba := image.NewRGBA(image.Rect(0, 0, int(metadata.SourceBounds.Width), int(metadata.SourceBounds.Height)))
	for y := 0; y < rgba.Bounds().Dy(); y++ {
		for x := 0; x < rgba.Bounds().Dx(); x++ {
			rgba.Set(x, y, decoded.At(x+windowRect.Min.X, y+windowRect.Min.Y))
		}
	}
	return sessions.DesktopActivationImage{Context: metadata, Image: rgba}, nil
}

func (b *Backend) captureActivation(ctx context.Context, _ bool) (sessions.DesktopActivationContext, error) {
	if b.platform != "windows" && b.platform != "darwin" {
		return sessions.DesktopActivationContext{}, ErrUnavailable
	}
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return sessions.DesktopActivationContext{}, err
		}
	}
	before, err := b.activationProbe(ctx)
	if err != nil {
		return sessions.DesktopActivationContext{}, err
	}
	if before.ActiveWindow == 0 || before.ProcessID == 0 || before.Width == 0 || before.Height == 0 {
		return sessions.DesktopActivationContext{}, ErrPermissionDenied
	}
	// A second identity read prevents publishing a focus transition or a
	// partially changed geometry as if it were one coherent observation.
	after, err := b.activationProbe(ctx)
	if err != nil || before.ActiveWindow != after.ActiveWindow || before.ProcessID != after.ProcessID || before.X != after.X || before.Y != after.Y || before.Width != after.Width || before.Height != after.Height {
		return sessions.DesktopActivationContext{}, ErrUnavailable
	}
	if b.session != "" {
		if err := b.CheckSession(ctx); err != nil {
			return sessions.DesktopActivationContext{}, err
		}
	}
	b.activationRootX, b.activationRootY = before.RootX, before.RootY
	now := time.Now().UTC()
	revision := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d/%d/%d", b.display, before.X, before.Y, before.Width, before.Height)))
	return sessions.DesktopActivationContext{
		SourceBounds: sessions.DesktopBounds{X: before.X, Y: before.Y, Width: before.Width, Height: before.Height},
		ActiveWindow: before.ActiveWindow, PointerWindow: before.PointerWindow, ProcessID: before.ProcessID,
		PointerX: before.PointerX, PointerY: before.PointerY, DisplayID: b.display,
		GeometryRevision: fmt.Sprintf("%x", revision[:]), CapturedAt: now,
	}, nil
}

// VerifyWindowProcess binds the companion's native identity to its process
// before and after activation capture. Windows can verify HWND ownership
// directly; macOS verifies the process through Accessibility and refuses an
// absent process rather than treating a nonzero pointer as proof.
func (b *Backend) VerifyWindowProcess(ctx context.Context, window uint64, pid uint32) error {
	if window == 0 || window > uint64(1<<63-1) || pid == 0 || (b.platform != "windows" && b.platform != "darwin") {
		return ErrUnavailable
	}
	var command []string
	if b.platform == "windows" {
		command = windowsVerifyWindowProcessCommand(window, pid)
	} else {
		command = macOSVerifyWindowProcessCommand(pid)
	}
	if err := b.runWithTimeout(ctx, command); err != nil {
		return ErrUnavailable
	}
	return nil
}

func (b *Backend) activationProbe(ctx context.Context) (activationRecord, error) {
	var command []string
	if b.platform == "windows" {
		command = windowsActivationProbeCommand()
	} else {
		command = macOSActivationProbeCommand()
	}
	data, err := b.commandWithTimeout(ctx, command)
	if err != nil || len(data) == 0 || len(data) > 16*1024 {
		return activationRecord{}, ErrUnavailable
	}
	var record activationRecord
	if json.Unmarshal(bytes.TrimSpace(data), &record) != nil {
		return activationRecord{}, ErrUnavailable
	}
	return record, nil
}

func (b *Backend) capture(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	if b.platform == "darwin" {
		if b.nativeCapture != nil {
			data, err := b.nativeCapture(ctx)
			if err == nil && len(data) > 0 {
				return data, nil
			}
			// Keep the existing command seam as a compatibility fallback for
			// older packaged helpers. A native capture failure is still surfaced
			// as permission evidence below when the fallback cannot produce data.
		}
		data, err := b.command(ctx, "screencapture", "-x", "-t", "png", "-")
		if err != nil || len(data) == 0 {
			// screencapture is present on supported macOS hosts; a failed
			// capture is therefore permission evidence rather than permission
			// inferred from executable discovery.
			return nil, fmt.Errorf("%w: screen recording capture failed: %v", ErrPermissionDenied, err)
		}
		return data, nil
	}
	args := windowsCaptureCommand()
	encoded, err := b.command(ctx, args[0], args[1:]...)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(encoded))
	if text == "" || len(text) > ((maxCaptureBytes+2)/3)*4 {
		return nil, ErrUnavailable
	}
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, ErrUnavailable
	}
	return data, nil
}

func (b *Backend) commandWithTimeout(ctx context.Context, args []string) ([]byte, error) {
	if len(args) == 0 || b.command == nil {
		return nil, ErrUnavailable
	}
	commandContext, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return b.command(commandContext, args[0], args[1:]...)
}

func (b *Backend) runWithTimeout(ctx context.Context, args []string) error {
	if len(args) == 0 || b.run == nil {
		return ErrUnavailable
	}
	commandContext, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return b.run(commandContext, args[0], args[1:]...)
}

func (b *Backend) validateAction(action *desktopv1.Action) error {
	if action == nil || (action.GetPointer() == nil && action.GetKey() == nil && action.GetWheel() == nil) {
		return ErrUnavailable
	}
	if p := action.GetPointer(); p != nil {
		if p.DisplayId != b.display || p.X < 0 || p.Y < 0 || p.X >= float64(b.width) || p.Y >= float64(b.height) || p.Kind < desktopv1.PointerAction_KIND_MOVE || p.Kind > desktopv1.PointerAction_KIND_CLICK || p.Button < desktopv1.PointerAction_BUTTON_UNSPECIFIED || p.Button > desktopv1.PointerAction_BUTTON_MIDDLE || (p.Kind != desktopv1.PointerAction_KIND_MOVE && p.Button == desktopv1.PointerAction_BUTTON_UNSPECIFIED) {
			return ErrUnavailable
		}
		if (b.platform == "windows" || b.platform == "darwin") && p.Kind != desktopv1.PointerAction_KIND_MOVE {
			held := b.heldButtons[p.Button]
			if p.Kind == desktopv1.PointerAction_KIND_UP && !held {
				return ErrUnavailable
			}
			if p.Kind == desktopv1.PointerAction_KIND_CLICK && held {
				return ErrUnavailable
			}
		}
	}
	if k := action.GetKey(); k != nil {
		if k.Kind < desktopv1.KeyAction_KIND_DOWN || k.Kind > desktopv1.KeyAction_KIND_PRESS || strings.TrimSpace(k.Key) == "" || len(k.Key) > 128 {
			return ErrUnavailable
		}
		heldID := heldKeyID(k.Key)
		if _, held := b.heldKeys[heldID]; held && k.Kind == desktopv1.KeyAction_KIND_PRESS {
			return ErrUnavailable
		}
		if k.Kind == desktopv1.KeyAction_KIND_DOWN {
			if _, ok := keyCode(b.platform, k.Key); !ok {
				return ErrUnavailable
			}
			// A repeated DOWN is idempotent for the helper-owned key.
		} else if k.Kind == desktopv1.KeyAction_KIND_UP {
			if _, ok := b.heldKeys[heldID]; !ok {
				return ErrUnavailable
			}
		}
	}
	if w := action.GetWheel(); w != nil {
		h, v := int64(w.HorizontalTicks), int64(w.VerticalTicks)
		if h < 0 {
			h = -h
		}
		if v < 0 {
			v = -v
		}
		if w.DisplayId != b.display || w.X < 0 || w.Y < 0 || w.X >= float64(b.width) || w.Y >= float64(b.height) || h+v == 0 || h+v > 20 {
			return ErrUnavailable
		}
	}
	return nil
}

func (b *Backend) apply(ctx context.Context, action *desktopv1.Action) error {
	if b.platform == "darwin" {
		return b.applyMacOS(ctx, action)
	}
	return b.applyWindows(ctx, action)
}

func (b *Backend) applyMacOS(ctx context.Context, action *desktopv1.Action) error {
	if pointer := action.GetPointer(); pointer != nil {
		return b.runWithTimeout(ctx, macOSPointerCommand(pointer))
	}
	var script string
	switch {
	case action.GetKey() != nil:
		script = fmt.Sprintf("tell application \"System Events\" to keystroke %s", appleScriptString(action.GetKey().Key))
	case action.GetWheel() != nil:
		script = macOSWheelScript(action.GetWheel())
	default:
		return ErrUnavailable
	}
	return b.run(ctx, "osascript", "-e", script)
}

func (b *Backend) applyWindows(ctx context.Context, action *desktopv1.Action) error {
	var expression string
	switch {
	case action.GetPointer() != nil:
		p := action.GetPointer()
		expression = windowsPointerExpression(strconv.Itoa(int(p.X)), strconv.Itoa(int(p.Y)), p.Kind, p.Button)
	case action.GetKey() != nil:
		expression = fmt.Sprintf("[System.Windows.Forms.SendKeys]::SendWait(%s)", powershellString(action.GetKey().Key))
	case action.GetWheel() != nil:
		expression = windowsWheelExpression(action.GetWheel())
	default:
		return ErrUnavailable
	}
	script := "Add-Type -AssemblyName System.Windows.Forms; Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VrooliInput { [DllImport(\"user32.dll\")] public static extern bool SetCursorPos(int X, int Y); [DllImport(\"user32.dll\")] public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extra); }'; " + expression
	return b.run(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func keyCode(platform, key string) (uint16, bool) {
	name := strings.ToUpper(strings.TrimSpace(key))
	if len([]rune(name)) == 1 {
		r := []rune(name)[0]
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return uint16(r), true
		}
	}
	if platform == "darwin" {
		codes := map[string]uint16{
			"RETURN": 36, "ENTER": 36, "TAB": 48, "SPACE": 49, "BACKSPACE": 51, "DELETE": 117, "ESC": 53, "ESCAPE": 53,
			"SHIFT": 56, "SHIFTLEFT": 56, "SHIFTRIGHT": 60, "CTRL": 59, "CONTROL": 59, "CTRLLEFT": 59, "CONTROLLEFT": 59, "CTRLRIGHT": 62, "CONTROLRIGHT": 62,
			"ALT": 58, "ALTLEFT": 58, "ALTRIGHT": 61, "META": 55, "METALEFT": 55, "METARIGHT": 54, "COMMAND": 55, "SUPER": 55,
			"LEFT": 123, "ARROWLEFT": 123, "RIGHT": 124, "ARROWRIGHT": 124, "DOWN": 125, "ARROWDOWN": 125, "UP": 126, "ARROWUP": 126, "HOME": 115, "END": 119, "PAGEUP": 116, "PAGEDOWN": 121,
			"F1": 122, "F2": 120, "F3": 99, "F4": 118, "F5": 96, "F6": 97, "F7": 98, "F8": 100, "F9": 101, "F10": 109, "F11": 103, "F12": 111,
		}
		code, ok := codes[name]
		return code, ok
	}
	codes := map[string]uint16{
		"RETURN": 0x0d, "ENTER": 0x0d, "TAB": 0x09, "SPACE": 0x20, "BACKSPACE": 0x08, "DELETE": 0x2e, "ESC": 0x1b, "ESCAPE": 0x1b,
		"SHIFT": 0x10, "SHIFTLEFT": 0xa0, "SHIFTRIGHT": 0xa1, "CTRL": 0x11, "CONTROL": 0x11, "CTRLLEFT": 0xa2, "CONTROLLEFT": 0xa2, "CTRLRIGHT": 0xa3, "CONTROLRIGHT": 0xa3,
		"ALT": 0x12, "ALTLEFT": 0xa4, "ALTRIGHT": 0xa5, "META": 0x5b, "METALEFT": 0x5b, "METARIGHT": 0x5c, "COMMAND": 0x5b, "SUPER": 0x5b,
		"LEFT": 0x25, "ARROWLEFT": 0x25, "UP": 0x26, "ARROWUP": 0x26, "RIGHT": 0x27, "ARROWRIGHT": 0x27, "DOWN": 0x28, "ARROWDOWN": 0x28, "HOME": 0x24, "END": 0x23, "PAGEUP": 0x21, "PAGEDOWN": 0x22,
		"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73, "F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77, "F9": 0x78, "F10": 0x79, "F11": 0x7a, "F12": 0x7b,
	}
	code, ok := codes[name]
	return code, ok
}

func heldKeyID(key string) string { return strings.ToUpper(strings.TrimSpace(key)) }

func windowsKeyCodeExpression(code uint16, down bool) string {
	flags := uint32(0)
	if !down {
		flags = 0x0002 // KEYEVENTF_KEYUP
	}
	return fmt.Sprintf("[VrooliKey]::keybd_event([byte]0x%02x, 0, [uint32]0x%04x, [UIntPtr]::Zero)", code, flags)
}

func macOSKeyCodeCommand(code uint16, down bool) []string {
	return []string{"osascript", "-l", "JavaScript", "-e", fmt.Sprintf("ObjC.import('CoreGraphics'); const event=$.CGEventCreateKeyboardEvent(null,%d,%t); if(event===null) throw new Error('keyboard event unavailable'); $.CGEventPost($.kCGHIDEventTap,event);", code, down)}
}

func macOSPointerCommand(pointer *desktopv1.PointerAction) []string {
	if pointer == nil {
		return nil
	}
	button := "$.kCGMouseButtonLeft"
	down, up := "$.kCGEventLeftMouseDown", "$.kCGEventLeftMouseUp"
	if pointer.Button == desktopv1.PointerAction_BUTTON_SECONDARY {
		button, down, up = "$.kCGMouseButtonRight", "$.kCGEventRightMouseDown", "$.kCGEventRightMouseUp"
	} else if pointer.Button == desktopv1.PointerAction_BUTTON_MIDDLE {
		button, down, up = "$.kCGMouseButtonCenter", "$.kCGEventOtherMouseDown", "$.kCGEventOtherMouseUp"
	}
	point := fmt.Sprintf("$.CGPointMake(%d,%d)", int(pointer.X), int(pointer.Y))
	if pointer.Kind == desktopv1.PointerAction_KIND_MOVE {
		return []string{"osascript", "-l", "JavaScript", "-e", fmt.Sprintf("ObjC.import('CoreGraphics'); $.CGWarpMouseCursorPosition(%s);", point)}
	}
	eventType := down
	if pointer.Kind == desktopv1.PointerAction_KIND_UP {
		eventType = up
	}
	return []string{"osascript", "-l", "JavaScript", "-e", fmt.Sprintf("ObjC.import('CoreGraphics'); const point=%s; $.CGWarpMouseCursorPosition(point); const event=$.CGEventCreateMouseEvent(null,%s,point,%s); if(event===null) throw new Error('mouse event unavailable'); $.CGEventPost($.kCGHIDEventTap,event);", point, eventType, button)}
}

func macOSReleaseButtonCommand(button desktopv1.PointerAction_Button) []string {
	if button == desktopv1.PointerAction_BUTTON_UNSPECIFIED {
		return nil
	}
	buttonName := "$.kCGMouseButtonLeft"
	eventType := "$.kCGEventLeftMouseUp"
	if button == desktopv1.PointerAction_BUTTON_SECONDARY {
		buttonName, eventType = "$.kCGMouseButtonRight", "$.kCGEventRightMouseUp"
	} else if button == desktopv1.PointerAction_BUTTON_MIDDLE {
		buttonName, eventType = "$.kCGMouseButtonCenter", "$.kCGEventOtherMouseUp"
	}
	return []string{"osascript", "-l", "JavaScript", "-e", fmt.Sprintf("ObjC.import('CoreGraphics'); const current=$.CGEventCreate(null); const point=$.CGEventGetLocation(current); const event=$.CGEventCreateMouseEvent(null,%s,point,%s); if(event===null) throw new Error('mouse release unavailable'); $.CGEventPost($.kCGHIDEventTap,event);", eventType, buttonName)}
}

func decodeAction(payload []byte) (*desktopv1.Action, error) {
	if len(payload) == 0 || len(payload) > 64*1024 {
		return nil, ErrUnavailable
	}
	var action desktopv1.Action
	if protojson.Unmarshal(payload, &action) != nil {
		return nil, ErrUnavailable
	}
	return &action, nil
}

func decodePNG(data []byte) (image.Image, error) {
	if len(data) == 0 || len(data) > maxCaptureBytes {
		return nil, ErrUnavailable
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxCapturePixels {
		return nil, ErrUnavailable
	}
	return png.Decode(bytes.NewReader(data))
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	var output boundedOutput
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func runCommandError(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func windowsCaptureCommand() []string {
	script := "$v=[System.Windows.Forms.SystemInformation]::VirtualScreen; $b=New-Object Drawing.Bitmap $v.Width,$v.Height; $g=[Drawing.Graphics]::FromImage($b); $g.CopyFromScreen($v.Left,$v.Top,0,0,$b.Size); $m=New-Object IO.MemoryStream; $b.Save($m,[Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($m.ToArray()); $g.Dispose(); $b.Dispose(); $m.Dispose()"
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; " + script}
}

func windowsActivationProbeCommand() []string {
	script := `$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.Windows.Forms; Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VrooliActivation { [StructLayout(LayoutKind.Sequential)] public struct P { public int X; public int Y; } [StructLayout(LayoutKind.Sequential)] public struct R { public int L; public int T; public int R; public int B; } [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow(); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint p); [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out R r); [DllImport("user32.dll")] public static extern bool GetCursorPos(out P p); }'; $h=[VrooliActivation]::GetForegroundWindow(); $p=[uint32]0; [VrooliActivation]::GetWindowThreadProcessId($h,[ref]$p)|Out-Null; $r=New-Object VrooliActivation+R; $c=New-Object VrooliActivation+P; if($h -eq [IntPtr]::Zero -or $p -eq 0 -or -not [VrooliActivation]::GetWindowRect($h,[ref]$r) -or -not [VrooliActivation]::GetCursorPos([ref]$c)){ throw 'foreground window unavailable' }; $v=[System.Windows.Forms.SystemInformation]::VirtualScreen; [pscustomobject]@{active_window=[uint64]$h.ToInt64();process_id=$p;pointer_window=[uint64]$h.ToInt64();pointer_x=[int32]$c.X;pointer_y=[int32]$c.Y;x=[int32]$r.L;y=[int32]$r.T;width=[uint32]($r.R-$r.L);height=[uint32]($r.B-$r.T);root_x=[int32]$v.Left;root_y=[int32]$v.Top} | ConvertTo-Json -Compress`
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func windowsVerifyWindowProcessCommand(window uint64, pid uint32) []string {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VrooliVerify { [DllImport("user32.dll")] public static extern bool IsWindow(IntPtr h); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint p); }'; $p=[uint32]0; $h=[IntPtr]::new([int64]%d); if(-not [VrooliVerify]::IsWindow($h) -or [VrooliVerify]::GetWindowThreadProcessId($h,[ref]$p) -eq 0 -or $p -ne [uint32]%d){ throw 'window process mismatch' }`, window, pid)
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func windowsApplicationsCommand() []string {
	script := `$ErrorActionPreference='Stop'; Add-Type -AssemblyName UIAutomationClient; Add-Type -AssemblyName UIAutomationTypes; $root=[System.Windows.Automation.AutomationElement]::RootElement; $windows=$root.FindAll([System.Windows.Automation.TreeScope]::Children,[System.Windows.Automation.Condition]::TrueCondition); $out=@(); foreach($w in $windows){ try { $c=$w.Current; if($c.ProcessId -gt 0 -and $c.ControlType.Id -eq 50032){ $out += [pscustomobject]@{name=[string]$c.Name; runtime_id=([string]::Join(',',($w.GetRuntimeId()))); process_id=[int]$c.ProcessId} } } catch {} }; if($out.Count -eq 0){ '[]' } else { $out | ConvertTo-Json -Compress -Depth 4 }`
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func windowsApplicationForProcessCommand(processID uint32) []string {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Add-Type -AssemblyName UIAutomationClient; Add-Type -AssemblyName UIAutomationTypes; $root=[System.Windows.Automation.AutomationElement]::RootElement; $windows=$root.FindAll([System.Windows.Automation.TreeScope]::Children,[System.Windows.Automation.Condition]::TrueCondition); $out=@(); foreach($w in $windows){ try { $c=$w.Current; if($c.ProcessId -eq %d -and $c.ControlType.Id -eq 50032){ $out += [pscustomobject]@{name=[string]$c.Name; runtime_id=([string]::Join(',',($w.GetRuntimeId()))); process_id=[int]$c.ProcessId} } } catch {} }; if($out.Count -eq 0){ '[]' } elseif($out.Count -eq 1){ $out | ConvertTo-Json -Compress -Depth 4 } else { $out | Select-Object -First 1 | ConvertTo-Json -Compress -Depth 4 }`, processID)
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func macOSApplicationsCommand() []string {
	script := `const se=Application("System Events"); const out=[]; for (const p of se.applicationProcesses()) { try { const pid=Number(p.unixId()); const name=String(p.name()); if (pid>0 && name) out.push({name:name,runtime_id:"pid:"+pid,process_id:pid}); } catch(e) {} } JSON.stringify(out);`
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func macOSActivationProbeCommand() []string {
	// System Events supplies a stable process/window identity and geometry in
	// the logged-in user's Accessibility session. Pointer coordinates are not
	// available through the noninteractive JXA bridge and remain zero evidence.
	script := `const se=Application("System Events"); const p=se.applicationProcesses().find(x=>{try{return x.frontmost()===true}catch(e){return false}}); if(!p) throw new Error("frontmost process unavailable"); const pid=Number(p.unixId()); const ws=p.windows(); if(!ws||ws.length===0) throw new Error("frontmost window unavailable"); const w=ws[0]; const pos=w.position(); const size=w.size(); JSON.stringify({active_window:pid,process_id:pid,pointer_window:pid,pointer_x:0,pointer_y:0,x:Number(pos[0]),y:Number(pos[1]),width:Number(size[0]),height:Number(size[1])});`
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func macOSVerifyWindowProcessCommand(pid uint32) []string {
	script := fmt.Sprintf(`const se=Application("System Events"); const found=se.applicationProcesses().some(p=>{try{if(Number(p.unixId())!==%d || p.frontmost()!==true) return false; const ws=p.windows(); return ws && ws.length>0}catch(e){return false}}); if(!found) throw new Error("frontmost window process unavailable");`, pid)
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func macOSApplicationForProcessCommand(processID uint32) []string {
	script := fmt.Sprintf(`const se=Application("System Events"); const pid=%d; const out=[]; for (const p of se.applicationProcesses()) { try { if (Number(p.unixId())===pid) { out.push({name:String(p.name()),runtime_id:"pid:"+pid,process_id:pid}); break; } } catch(e) {} } JSON.stringify(out);`, processID)
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func macOSSemanticCommand(runtimeID string) []string {
	pid, _, _ := macOSRuntimeParts(runtimeID)
	script := fmt.Sprintf(`const se=Application("System Events"); const pid=%d; const out=[]; let proc=null; for (const p of se.applicationProcesses()) { try { if (Number(p.unixId())===pid) { proc=p; break; } } catch(e) {} } function str(v){ try { return v==null ? "" : String(v); } catch(e) { return ""; } } function add(e,parent,windowNumber,id,rootIndex){ let role=""; try { role=str(e.role()); } catch(err) {} let value=""; try { value=str(e.value()); } catch(err) {} const editable=role.indexOf("TextField")>=0 || role.indexOf("TextArea")>=0 || role.indexOf("ComboBox")>=0; out.push({name:str(e.name()),runtime_id:"pid:"+pid+":"+windowNumber+":"+id,value:value,editable:editable,role:0,parent_index:parent,window_index:rootIndex}); } if(proc){ let windows=[]; try { windows=proc.windows(); } catch(e) {} for(let wi=0;wi<windows.length && out.length<128;wi++){ const rootIndex=out.length; add(windows[wi],-1,wi,0,rootIndex); let children=[]; try { children=windows[wi].entireContents(); } catch(e) {} for(let ei=0;ei<children.length && out.length<128;ei++) add(children[ei],rootIndex,wi,ei+1,rootIndex); } } JSON.stringify(out);`, pid)
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func windowsSemanticCommand(runtimeID string) []string {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Add-Type -AssemblyName UIAutomationClient; Add-Type -AssemblyName UIAutomationTypes; $rid=[int[]]@(%s); $window=[System.Windows.Automation.AutomationElement]::FromRuntimeId($rid); if($null -eq $window){ throw 'window not found' }; $out=New-Object System.Collections.Generic.List[object]; function Walk([System.Windows.Automation.AutomationElement]$e,[int]$parent,[int]$depth){ if($out.Count -ge 128 -or $depth -gt 16){ return }; try { $c=$e.Current; $id=$out.Count; $editable=$false; $pattern=$null; try { $editable=$e.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern,[ref]$pattern) } catch {}; $value=''; if($editable){ try { $value=[string]$pattern.Current.Value } catch {} }; $out.Add([pscustomobject]@{name=[string]$c.Name; runtime_id=([string]::Join(',',($e.GetRuntimeId()))); value=$value; editable=[bool]$editable; role=[int]$c.ControlType.Id; parent_index=$parent; window_index=0}); $children=$e.FindAll([System.Windows.Automation.TreeScope]::Children,[System.Windows.Automation.Condition]::TrueCondition); foreach($child in $children){ Walk $child $id ($depth+1) } } catch {} }; Walk $window -1 0; if($out.Count -eq 0){ '[]' } else { $out | ConvertTo-Json -Compress -Depth 5 }`, runtimeID)
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func windowsReadTextCommand(runtimeID string) []string {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Add-Type -AssemblyName UIAutomationClient; Add-Type -AssemblyName UIAutomationTypes; $rid=[int[]]@(%s); $e=[System.Windows.Automation.AutomationElement]::FromRuntimeId($rid); if($null -eq $e){ throw 'element not found' }; $p=$null; if(-not $e.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern,[ref]$p)){ throw 'value pattern unavailable' }; ([string]$p.Current.Value) | ConvertTo-Json -Compress`, runtimeID)
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func windowsSetTextCommand(runtimeID, value string) []string {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Add-Type -AssemblyName UIAutomationClient; Add-Type -AssemblyName UIAutomationTypes; $rid=[int[]]@(%s); $e=[System.Windows.Automation.AutomationElement]::FromRuntimeId($rid); if($null -eq $e){ throw 'element not found' }; $p=$null; if(-not $e.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern,[ref]$p)){ throw 'value pattern unavailable' }; $p.SetValue(%s)`, runtimeID, powershellString(value))
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func windowsInvokeCommand(runtimeID string) []string {
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Add-Type -AssemblyName UIAutomationClient; Add-Type -AssemblyName UIAutomationTypes; $rid=[int[]]@(%s); $e=[System.Windows.Automation.AutomationElement]::FromRuntimeId($rid); if($null -eq $e){ throw 'element not found' }; $p=$null; if(-not $e.TryGetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern,[ref]$p)){ throw 'invoke pattern unavailable' }; $p.Invoke()`, runtimeID)
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script}
}

func (b *Backend) readSemanticText(ctx context.Context, runtimeID string) (string, error) {
	if !b.validRuntimeID(runtimeID) {
		return "", ErrUnavailable
	}
	command := windowsReadTextCommand(runtimeID)
	if b.platform == "darwin" {
		command = macOSReadTextCommand(runtimeID)
	}
	data, err := b.commandWithTimeout(ctx, command)
	if err != nil || len(data) > 64*1024 {
		return "", ErrUnavailable
	}
	var value string
	if json.Unmarshal(bytes.TrimSpace(data), &value) != nil {
		return "", ErrUnavailable
	}
	return value, nil
}

func (b *Backend) setSemanticText(ctx context.Context, runtimeID, value string) error {
	if len(value) > 64*1024 {
		return ErrUnavailable
	}
	if !b.validRuntimeID(runtimeID) {
		return ErrUnavailable
	}
	command := windowsSetTextCommand(runtimeID, value)
	if b.platform == "darwin" {
		command = macOSSetTextCommand(runtimeID, value)
	}
	return b.runWithTimeout(ctx, command)
}

func (b *Backend) invokeSemantic(ctx context.Context, runtimeID string) error {
	if !b.validRuntimeID(runtimeID) {
		return ErrUnavailable
	}
	command := windowsInvokeCommand(runtimeID)
	if b.platform == "darwin" {
		command = macOSInvokeCommand(runtimeID)
	}
	return b.runWithTimeout(ctx, command)
}

// verifySemanticProcessLocked rechecks the process that produced the cached
// semantic tree immediately before an action. Runtime IDs are only meaningful
// while their process remains alive; a PID/session check prevents a stale
// element from being applied to a replacement window after process exit.
func (b *Backend) verifySemanticProcessLocked(ctx context.Context) error {
	if b.semanticProcessID == 0 || (b.platform != "windows" && b.platform != "darwin") {
		return ErrUnavailable
	}
	command := windowsApplicationForProcessCommand(b.semanticProcessID)
	if b.platform == "darwin" {
		command = macOSApplicationForProcessCommand(b.semanticProcessID)
	}
	data, err := b.commandWithTimeout(ctx, command)
	if err != nil {
		return ErrUnavailable
	}
	records, err := decodeWindowsApplications(data)
	if err != nil || len(records) != 1 || records[0].ProcessID != b.semanticProcessID || !b.validRuntimeID(records[0].RuntimeID) {
		return ErrUnavailable
	}
	return nil
}

// macOS runtime IDs are helper-generated ordinals because Accessibility does
// not expose a portable element identity. Re-enumerate the current tree and
// compare the selected element's identity and ancestry before any action. A
// surviving PID alone is insufficient: a window can close or reorder while
// the application remains alive.
func (b *Backend) verifySemanticElementLocked(ctx context.Context, elementID string) error {
	if b.platform != "darwin" {
		return nil
	}
	element, ok := b.elements[elementID]
	if !ok {
		return ErrUnavailable
	}
	pid, _, _, ok := macOSRuntimePartsOK(element.runtimeID)
	if !ok {
		return ErrUnavailable
	}
	data, err := b.commandWithTimeout(ctx, macOSSemanticCommand(fmt.Sprintf("pid:%d", pid)))
	if err != nil {
		return ErrUnavailable
	}
	records, err := decodeWindowsElements(data)
	if err != nil || len(records) == 0 || len(records) > 128 {
		return ErrUnavailable
	}
	byRuntime := make(map[string]windowsElementRecord, len(records))
	for _, record := range records {
		if !b.validRuntimeID(record.RuntimeID) {
			return ErrUnavailable
		}
		byRuntime[record.RuntimeID] = record
	}
	record, ok := byRuntime[element.runtimeID]
	if !ok || record.Name != element.name || record.Editable != element.editable || roleFromRuntimeRecord(record) != element.role {
		return ErrUnavailable
	}
	if element.parentID != "" {
		parent, ok := b.elements[element.parentID]
		if !ok || record.ParentIndex < 0 || record.ParentIndex >= len(records) || records[record.ParentIndex].RuntimeID != parent.runtimeID {
			return ErrUnavailable
		}
	} else if record.ParentIndex != -1 {
		return ErrUnavailable
	}
	if element.windowID != "" {
		window, ok := b.elements[element.windowID]
		if !ok || record.WindowIndex < 0 || record.WindowIndex >= len(records) || records[record.WindowIndex].RuntimeID != window.runtimeID {
			return ErrUnavailable
		}
	}
	return nil
}

func macOSReadTextCommand(runtimeID string) []string {
	pid, window, element := macOSRuntimeParts(runtimeID)
	script := fmt.Sprintf(`const se=Application("System Events"); const pid=%d; const wi=%d; const ei=%d; let target=null; for (const p of se.applicationProcesses()) { try { if (Number(p.unixId())===pid) { const ws=p.windows(); if(wi>=0 && wi<ws.length){ target=ws[wi]; if(ei>0){ const all=target.entireContents(); target=all[ei-1]; } } break; } } catch(e) {} } let value=""; try { if(target) value=String(target.value()); } catch(e) {} JSON.stringify(value);`, pid, window, element)
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func macOSSetTextCommand(runtimeID, value string) []string {
	pid, window, element := macOSRuntimeParts(runtimeID)
	script := fmt.Sprintf(`const se=Application("System Events"); const pid=%d; const wi=%d; const ei=%d; const value=%s; let target=null; for (const p of se.applicationProcesses()) { try { if (Number(p.unixId())===pid) { const ws=p.windows(); if(wi>=0 && wi<ws.length){ target=ws[wi]; if(ei>0){ const all=target.entireContents(); target=all[ei-1]; } } break; } } catch(e) {} } if(!target) throw new Error("element not found"); target.value=value;`, pid, window, element, strconv.Quote(value))
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func macOSInvokeCommand(runtimeID string) []string {
	pid, window, element := macOSRuntimeParts(runtimeID)
	script := fmt.Sprintf(`const se=Application("System Events"); const pid=%d; const wi=%d; const ei=%d; let target=null; for (const p of se.applicationProcesses()) { try { if (Number(p.unixId())===pid) { const ws=p.windows(); if(wi>=0 && wi<ws.length){ target=ws[wi]; if(ei>0){ const all=target.entireContents(); target=all[ei-1]; } } break; } } catch(e) {} } if(!target) throw new Error("element not found"); target.click();`, pid, window, element)
	return []string{"osascript", "-l", "JavaScript", "-e", script}
}

func decodeWindowsApplications(data []byte) ([]windowsApplicationRecord, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || len(trimmed) > 128*1024 {
		return nil, ErrUnavailable
	}
	if trimmed[0] == '{' {
		var record windowsApplicationRecord
		if err := json.Unmarshal(trimmed, &record); err != nil {
			return nil, ErrUnavailable
		}
		return []windowsApplicationRecord{record}, nil
	}
	var records []windowsApplicationRecord
	if err := json.Unmarshal(trimmed, &records); err != nil {
		return nil, ErrUnavailable
	}
	return records, nil
}

func decodeWindowsElements(data []byte) ([]windowsElementRecord, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || len(trimmed) > 512*1024 {
		return nil, ErrUnavailable
	}
	if trimmed[0] == '{' {
		var record windowsElementRecord
		if err := json.Unmarshal(trimmed, &record); err != nil {
			return nil, ErrUnavailable
		}
		return []windowsElementRecord{record}, nil
	}
	var records []windowsElementRecord
	if err := json.Unmarshal(trimmed, &records); err != nil {
		return nil, ErrUnavailable
	}
	return records, nil
}

func validRuntimeID(value string) bool {
	parts := strings.Split(value, ",")
	if len(parts) == 0 || len(parts) > 32 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || parsed < 0 {
			return false
		}
	}
	return true
}

func (b *Backend) validRuntimeID(value string) bool {
	if b.platform != "darwin" {
		return validRuntimeID(value)
	}
	_, _, _, ok := macOSRuntimePartsOK(value)
	return ok
}

func macOSRuntimeParts(value string) (pid, window, element int) {
	pid, window, element, _ = macOSRuntimePartsOK(value)
	return pid, window, element
}

func macOSRuntimePartsOK(value string) (pid, window, element int, ok bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 4 || parts[0] != "pid" {
		return 0, 0, 0, false
	}
	parsed := make([]int, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		v, err := strconv.Atoi(parts[i])
		if err != nil || v < 0 {
			return 0, 0, 0, false
		}
		parsed[i-1] = v
	}
	if parsed[0] == 0 {
		return 0, 0, 0, false
	}
	if len(parsed) == 1 {
		return parsed[0], 0, 0, true
	}
	return parsed[0], parsed[1], parsed[2], true
}

func roleFromRuntimeRecord(record windowsElementRecord) uint32 {
	if record.Role != 0 {
		return record.Role
	}
	if record.Editable {
		return 50004 // ControlType.Edit, for older helper output.
	}
	return 0
}

func windowsPointerExpression(x, y string, kind desktopv1.PointerAction_Kind, button desktopv1.PointerAction_Button) string {
	flags := map[desktopv1.PointerAction_Button]string{desktopv1.PointerAction_BUTTON_PRIMARY: "0x0002", desktopv1.PointerAction_BUTTON_SECONDARY: "0x0008", desktopv1.PointerAction_BUTTON_MIDDLE: "0x0020"}
	ups := map[desktopv1.PointerAction_Button]string{desktopv1.PointerAction_BUTTON_PRIMARY: "0x0004", desktopv1.PointerAction_BUTTON_SECONDARY: "0x0010", desktopv1.PointerAction_BUTTON_MIDDLE: "0x0040"}
	flag, up := flags[button], ups[button]
	if flag == "" {
		flag, up = flags[desktopv1.PointerAction_BUTTON_PRIMARY], ups[desktopv1.PointerAction_BUTTON_PRIMARY]
	}
	move := fmt.Sprintf("[VrooliInput]::SetCursorPos(%s,%s)", x, y)
	switch kind {
	case desktopv1.PointerAction_KIND_DOWN:
		return move + fmt.Sprintf("; [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", flag)
	case desktopv1.PointerAction_KIND_UP:
		return move + fmt.Sprintf("; [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", up)
	default:
		return move + fmt.Sprintf("; [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero); [VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", flag, up)
	}
}

func (b *Backend) releaseWindowsButton(ctx context.Context, button desktopv1.PointerAction_Button) error {
	ups := map[desktopv1.PointerAction_Button]string{desktopv1.PointerAction_BUTTON_PRIMARY: "0x0004", desktopv1.PointerAction_BUTTON_SECONDARY: "0x0010", desktopv1.PointerAction_BUTTON_MIDDLE: "0x0040"}
	up, ok := ups[button]
	if !ok {
		return ErrUnavailable
	}
	script := "Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VrooliInput { [DllImport(\"user32.dll\")] public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extra); }'; " + fmt.Sprintf("[VrooliInput]::mouse_event(%s,0,0,0,[UIntPtr]::Zero)", up)
	return b.runWithTimeout(ctx, []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script})
}

func windowsWheelExpression(action *desktopv1.WheelAction) string {
	if action == nil {
		return ""
	}
	move := fmt.Sprintf("[VrooliInput]::SetCursorPos(%d,%d)", int(action.X), int(action.Y))
	parts := []string{move}
	if action.VerticalTicks != 0 {
		parts = append(parts, fmt.Sprintf("[VrooliInput]::mouse_event(0x0800,0,0,%d,[UIntPtr]::Zero)", int(action.VerticalTicks)*120))
	}
	if action.HorizontalTicks != 0 {
		parts = append(parts, fmt.Sprintf("[VrooliInput]::mouse_event(0x1000,0,0,%d,[UIntPtr]::Zero)", int(action.HorizontalTicks)*120))
	}
	return strings.Join(parts, "; ")
}

func macOSWheelScript(action *desktopv1.WheelAction) string {
	if action == nil {
		return ""
	}
	var commands []string
	appendScroll := func(direction string, ticks int32) {
		if ticks < 0 {
			direction = oppositeScrollDirection(direction)
			ticks = -ticks
		}
		for i := int32(0); i < ticks; i++ {
			commands = append(commands, "scroll "+direction)
		}
	}
	appendScroll("up", action.VerticalTicks)
	appendScroll("right", action.HorizontalTicks)
	return "tell application \"System Events\"\n" + strings.Join(commands, "\n") + "\nend tell"
}

func oppositeScrollDirection(direction string) string {
	switch direction {
	case "up":
		return "down"
	case "right":
		return "left"
	default:
		return direction
	}
}

func powershellString(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
