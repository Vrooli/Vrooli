package desktophelper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"device-control/internal/native/atspi"
	"device-control/internal/sessions"
	"github.com/google/uuid"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

type semanticPixels interface {
	sessions.DesktopNative
	sessions.DesktopObserver
	CheckSession(context.Context) error
}
type semanticAccess interface {
	ProcessRoot(context.Context, string, uint32) (atspi.Ref, error)
	Children(context.Context, atspi.Ref) ([]atspi.Ref, error)
	Name(context.Context, atspi.Ref) (string, error)
	Description(context.Context, atspi.Ref) (string, error)
	Role(context.Context, atspi.Ref) (uint32, error)
	State(context.Context, atspi.Ref) ([]string, error)
	Bounds(context.Context, atspi.Ref) (int32, int32, uint32, uint32, bool, error)
	Actions(context.Context, atspi.Ref) ([]string, bool, error)
	Editable(context.Context, atspi.Ref) (bool, error)
	ReadText(context.Context, atspi.Ref) (string, error)
	InsertTextChecked(context.Context, atspi.Ref, string, int32, string, func(context.Context) error) error
	Focus(context.Context, atspi.Ref) error
	Invoke(context.Context, atspi.Ref) error
	InvokeNamed(context.Context, atspi.Ref, string) error
}
type semanticEntry struct {
	children         []atspi.Ref
	ref              atspi.Ref
	text             string
	editable         bool
	supportedActions []string
}
type semanticBackend struct {
	observed                              []sessions.DesktopSemanticElement
	pixels                                semanticPixels
	access                                semanticAccess
	busID                                 string
	mu                                    sync.Mutex
	revision, leaseID, geometry           string
	expires                               time.Time
	entries                               map[string]semanticEntry
	applications                          map[string]atspi.Ref
	applicationRevision, applicationLease string
	applicationExpires                    time.Time
	refreshEpoch                          uint64
}

func (b *semanticBackend) clearApplications() {
	b.applications = nil
	b.applicationRevision = ""
	b.applicationLease = ""
}

func (b *semanticBackend) Applications(ctx context.Context, leaseID string) (sessions.DesktopApplications, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clear()
	b.clearApplications()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	access, ok := b.access.(interface {
		Applications(context.Context, string) ([]atspi.Application, error)
	})
	if !ok || leaseID == "" {
		return sessions.DesktopApplications{}, atspi.ErrRefused
	}
	if err := b.pixels.CheckSession(ctx); err != nil {
		return sessions.DesktopApplications{}, err
	}
	apps, err := access.Applications(ctx, b.busID)
	if err != nil {
		return sessions.DesktopApplications{}, err
	}
	if err := b.pixels.CheckSession(ctx); err != nil {
		return sessions.DesktopApplications{}, err
	}
	catalog := sessions.DesktopApplications{Revision: uuid.NewString(), ExpiresAt: time.Now().Add(30 * time.Second)}
	refs := map[string]atspi.Ref{}
	for _, app := range apps {
		id := uuid.NewString()
		refs[id] = app.Ref
		catalog.Applications = append(catalog.Applications, sessions.DesktopApplication{ID: id, Name: app.Name, ProcessID: app.Ref.PID})
	}
	b.applications, b.applicationRevision, b.applicationLease, b.applicationExpires = refs, catalog.Revision, leaseID, catalog.ExpiresAt
	return catalog, nil
}

func (b *semanticBackend) ObserveApplication(ctx context.Context, id, revision, leaseID string) (sessions.DesktopObservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clear()
	root, ok := b.applications[id]
	if !ok || leaseID == "" || leaseID != b.applicationLease || revision != b.applicationRevision || !time.Now().Before(b.applicationExpires) {
		return sessions.DesktopObservation{}, atspi.ErrRefused
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// Traverse the retained unique owner. Never resolve a replacement by PID.
	return b.observeRoot(ctx, root, leaseID)
}

func (b *semanticBackend) clear() { b.observed = nil; b.entries = nil; b.revision = ""; b.leaseID = "" }
func (b *semanticBackend) Observe(ctx context.Context) (sessions.DesktopObservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clear()
	return b.pixels.Observe(ctx)
}

func (b *semanticBackend) ObserveProcess(ctx context.Context, pid uint32, leaseID string) (sessions.DesktopObservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clear()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	root, err := b.access.ProcessRoot(ctx, b.busID, pid)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	return b.observeRoot(ctx, root, leaseID)
}

// Caller holds mu and supplies a bounded context.
func (b *semanticBackend) observeRoot(ctx context.Context, root atspi.Ref, leaseID string) (sessions.DesktopObservation, error) {
	snapshot, err := b.pixels.Observe(ctx)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	type node struct {
		ref            atspi.Ref
		parent, window string
	}
	queue := []node{{ref: root}}
	seen := map[atspi.Ref]bool{}
	entries := map[string]semanticEntry{}
	b.refreshEpoch++
	if b.refreshEpoch == 0 {
		b.refreshEpoch = 1
	}
	observation := &sessions.DesktopSemanticObservation{Revision: uuid.NewString(), ProcessID: root.PID, RefreshEpoch: b.refreshEpoch}
	textBytes := 0
	for len(queue) > 0 {
		if len(seen) >= 128 {
			return sessions.DesktopObservation{}, atspi.ErrRefused
		}
		item := queue[0]
		ref := item.ref
		queue = queue[1:]
		if seen[ref] {
			return sessions.DesktopObservation{}, atspi.ErrRefused
		}
		seen[ref] = true
		name, err := b.access.Name(ctx, ref)
		if err != nil {
			return sessions.DesktopObservation{}, err
		}
		label, err := b.access.Description(ctx, ref)
		if err != nil {
			label = ""
		}
		editable, err := b.access.Editable(ctx, ref)
		if err != nil {
			return sessions.DesktopObservation{}, err
		}
		entry := semanticEntry{ref: ref, editable: editable}
		if editable {
			entry.text, err = b.access.ReadText(ctx, ref)
			if err != nil {
				return sessions.DesktopObservation{}, err
			}
			textBytes += len(entry.text)
			if textBytes > 64*1024 {
				return sessions.DesktopObservation{}, atspi.ErrRefused
			}
		}
		id := uuid.NewString()
		role, err := b.access.Role(ctx, ref)
		if err != nil {
			return sessions.DesktopObservation{}, err
		}
		states, err := b.access.State(ctx, ref)
		stateKnown := err == nil
		if err != nil {
			states = nil
		}
		x, y, width, height, boundsKnown, err := b.access.Bounds(ctx, ref)
		if err != nil {
			x, y, width, height, boundsKnown = 0, 0, 0, 0, false
		}
		actions, _, err := b.access.Actions(ctx, ref)
		if err != nil {
			actions = nil
		}
		entry.supportedActions = slices.Clone(actions)
		window := item.window
		if role == 23 || role == 16 || role == 69 {
			window = id
		}
		entries[id] = entry
		if label == "" {
			label = name
		}
		observation.Elements = append(observation.Elements, sessions.DesktopSemanticElement{ID: id, Name: name, Label: label, Editable: editable, ParentID: item.parent, WindowID: window, Role: role, States: states, X: x, Y: y, Width: width, Height: height, SupportedActions: actions, BoundsKnown: boundsKnown, StateKnown: stateKnown, Fingerprint: semanticFingerprint(ref)})
		children, err := b.access.Children(ctx, ref)
		if err != nil {
			return sessions.DesktopObservation{}, err
		}
		entry.children = slices.Clone(children)
		entries[id] = entry
		for _, child := range children {
			queue = append(queue, node{ref: child, parent: id, window: window})
		}
		if len(queue) > 256 {
			return sessions.DesktopObservation{}, atspi.ErrRefused
		}
	}
	if err := b.pixels.CheckSession(ctx); err != nil {
		return sessions.DesktopObservation{}, err
	}
	observation.ExpiresAt = time.Now().Add(5 * time.Second)
	b.entries = entries
	b.revision = observation.Revision
	b.leaseID = leaseID
	b.geometry = snapshot.GeometryRevision
	b.expires = observation.ExpiresAt
	b.observed = observation.Elements
	snapshot.Semantic = observation
	return snapshot, nil
}

func (b *semanticBackend) text(command sessions.DesktopCommand) (*desktopv1.TextAction, semanticEntry, error) {
	var action desktopv1.Action
	if protojson.Unmarshal(command.Payload, &action) != nil {
		return nil, semanticEntry{}, atspi.ErrRefused
	}
	text := action.GetText()
	if check := action.GetAssertText(); check != nil {
		text = &desktopv1.TextAction{Text: check.ExpectedText, ElementId: check.ElementId, ObservationRevision: check.ObservationRevision}
	}
	if text == nil {
		return nil, semanticEntry{}, nil
	}
	entry, ok := b.entries[text.ElementId]
	if !ok || !entry.editable || text.ObservationRevision != b.revision || command.Lease.Ref.SessionID != b.leaseID || command.GeometryRevision != b.geometry || !time.Now().Before(b.expires) || text.Position < 0 || int64(text.Position) > int64(utf8.RuneCountInString(entry.text)) {
		return nil, semanticEntry{}, atspi.ErrRefused
	}
	return text, entry, nil
}

func (b *semanticBackend) invoke(command sessions.DesktopCommand) (semanticEntry, string, error) {
	var action desktopv1.Action
	if protojson.Unmarshal(command.Payload, &action) != nil {
		return semanticEntry{}, "", atspi.ErrRefused
	}
	invoke := action.GetInvoke()
	if invoke == nil {
		return semanticEntry{}, "", nil
	}
	entry, ok := b.entries[invoke.ElementId]
	if !ok || invoke.ObservationRevision != b.revision || command.Lease.Ref.SessionID != b.leaseID || command.GeometryRevision != b.geometry || !time.Now().Before(b.expires) {
		return semanticEntry{}, "", atspi.ErrRefused
	}
	if invoke.ActionName != "" && !semanticActionSupported(entry.supportedActions, invoke.ActionName) {
		return semanticEntry{}, "", atspi.ErrRefused
	}
	return entry, invoke.ActionName, nil
}

func semanticActionSupported(actions []string, wanted string) bool {
	for _, action := range actions {
		if strings.EqualFold(action, wanted) {
			return true
		}
	}
	return false
}

func (b *semanticBackend) Validate(ctx context.Context, command sessions.DesktopCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	text, entry, err := b.text(command)
	if err != nil {
		return err
	}
	invokeEntry, _, invokeErr := b.invoke(command)
	if invokeErr != nil {
		return invokeErr
	}
	if invokeEntry.ref != (atspi.Ref{}) {
		if err := b.pixels.CheckSession(ctx); err != nil {
			return err
		}
		return nil
	}
	if text == nil {
		return b.pixels.Validate(ctx, command)
	}
	if err := b.pixels.CheckSession(ctx); err != nil {
		return err
	}
	current, err := b.access.ReadText(ctx, entry.ref)
	if err != nil || current != entry.text {
		return atspi.ErrRefused
	}
	if isTextAssertion(command) && current != text.Text {
		return atspi.ErrRefused
	}
	return nil
}

func isTextAssertion(command sessions.DesktopCommand) bool {
	var action desktopv1.Action
	return protojson.Unmarshal(command.Payload, &action) == nil && action.GetAssertText() != nil
}

func (b *semanticBackend) Apply(ctx context.Context, command sessions.DesktopCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	text, entry, err := b.text(command)
	if err != nil {
		return err
	}
	invokeEntry, actionName, invokeErr := b.invoke(command)
	if invokeErr != nil {
		return invokeErr
	}
	if invokeEntry.ref != (atspi.Ref{}) {
		if err := b.pixels.CheckSession(ctx); err != nil {
			return err
		}
		if err := b.access.Focus(ctx, invokeEntry.ref); err != nil {
			return err
		}
		b.clear()
		if actionName != "" {
			return b.access.InvokeNamed(ctx, invokeEntry.ref, actionName)
		}
		return b.access.Invoke(ctx, invokeEntry.ref)
	}
	if text == nil {
		return b.pixels.Apply(ctx, command)
	}
	if err := b.pixels.CheckSession(ctx); err != nil {
		return err
	}
	if err := b.access.Focus(ctx, entry.ref); err != nil {
		return err
	}
	editable, err := b.access.Editable(ctx, entry.ref)
	if err != nil || !editable {
		return atspi.ErrRefused
	}
	b.clear() // Never reuse an observation after an attempted mutation.
	if isTextAssertion(command) {
		current, err := b.access.ReadText(ctx, entry.ref)
		if err != nil || current != entry.text || current != text.Text {
			return atspi.ErrRefused
		}
		return sessions.CheckDesktopMutation(ctx)
	}
	return b.access.InsertTextChecked(ctx, entry.ref, entry.text, text.Position, text.Text, sessions.CheckDesktopMutation)
}

func (b *semanticBackend) ReleaseHeld(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clear()
	b.clearApplications()
	return b.pixels.ReleaseHeld(ctx)
}
func (b *semanticBackend) CheckSession(ctx context.Context) error { return b.pixels.CheckSession(ctx) }

// Resolve consumes only helper-owned observation references; it never chooses
// the first candidate or resolves a replacement application by name or PID.
func (b *semanticBackend) Resolve(ctx context.Context, selector sessions.DesktopSelector, leaseID string) (sessions.DesktopResolution, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if leaseID == "" || leaseID != b.leaseID || selector.Revision != b.revision || (selector.RefreshEpoch != 0 && selector.RefreshEpoch != b.refreshEpoch) || !time.Now().Before(b.expires) || selector.Name == "" {
		return sessions.DesktopResolution{}, atspi.ErrRefused
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := b.pixels.CheckSession(ctx); err != nil {
		return sessions.DesktopResolution{}, err
	}
	windowFound := false
	for _, element := range b.observed {
		if element.ID == selector.WindowID && element.WindowID == element.ID {
			windowFound = true
		}
	}
	if !windowFound {
		return sessions.DesktopResolution{}, atspi.ErrRefused
	}
	result := sessions.DesktopResolution{Disposition: "absent", Revision: b.revision, GeometryRevision: b.geometry, ExpiresAt: b.expires}
	for _, element := range b.observed {
		if element.WindowID != selector.WindowID {
			continue
		}
		entry, ok := b.entries[element.ID]
		if !ok {
			return sessions.DesktopResolution{}, atspi.ErrRefused
		}
		name, err := b.access.Name(ctx, entry.ref)
		if err != nil || name != element.Name {
			return sessions.DesktopResolution{}, atspi.ErrRefused
		}
		if element.Fingerprint != "" && semanticFingerprint(entry.ref) != element.Fingerprint {
			return sessions.DesktopResolution{}, atspi.ErrRefused
		}
		children, err := b.access.Children(ctx, entry.ref)
		if err != nil || !slices.Equal(children, entry.children) {
			return sessions.DesktopResolution{}, atspi.ErrRefused
		}
		role, err := b.access.Role(ctx, entry.ref)
		if err != nil || role != element.Role {
			return sessions.DesktopResolution{}, atspi.ErrRefused
		}
		editable, err := b.access.Editable(ctx, entry.ref)
		if err != nil || editable != element.Editable {
			return sessions.DesktopResolution{}, atspi.ErrRefused
		}
		if element.Role != selector.Role && selector.Role != 0 {
			continue
		}
		if !semanticElementAllowed(element, selector) {
			continue
		}
		if semanticNameMatches(element.Name, selector.Name, selector.MatchMode) && (!selector.EditableOnly || element.Editable) {
			result.ElementIDs = append(result.ElementIDs, element.ID)
		}
	}
	if err := b.pixels.CheckSession(ctx); err != nil {
		return sessions.DesktopResolution{}, err
	}
	if ctx.Err() != nil || !time.Now().Before(b.expires) {
		return sessions.DesktopResolution{}, atspi.ErrRefused
	}
	if len(result.ElementIDs) == 1 {
		result.Disposition = "unique"
	} else if len(result.ElementIDs) > 1 {
		result.Disposition = "ambiguous"
	}
	return result, nil
}

func semanticElementAllowed(element sessions.DesktopSemanticElement, selector sessions.DesktopSelector) bool {
	if !selector.AllowHidden && element.StateKnown && len(element.States) > 0 && !hasSemanticState(element.States, "visible", "showing") {
		return false
	}
	if !selector.AllowDisabled && element.StateKnown && len(element.States) > 0 && !hasSemanticState(element.States, "enabled", "sensitive") {
		return false
	}
	if !selector.AllowOffscreen && element.BoundsKnown && (element.Width == 0 || element.Height == 0) {
		return false
	}
	return true
}

func hasSemanticState(states []string, wanted ...string) bool {
	for _, state := range states {
		for _, candidate := range wanted {
			if strings.EqualFold(state, candidate) {
				return true
			}
		}
	}
	return false
}

func semanticFingerprint(ref atspi.Ref) string {
	digest := sha256.Sum256([]byte(ref.Owner + "\x00" + string(ref.Path)))
	return hex.EncodeToString(digest[:])
}

// semanticNameMatches provides bounded, explicit matching for accessibility
// names. Fuzzy matching is deliberately conservative: it is limited to one
// observed window (the caller supplies that scope), optional role/editability
// constraints, and a small rune edit-distance budget.
func semanticNameMatches(candidate, wanted string, mode sessions.SemanticMatchMode) bool {
	switch mode {
	case sessions.SemanticMatchExact:
		return candidate == wanted
	case sessions.SemanticMatchNormalized:
		return normalizeSemanticName(candidate) == normalizeSemanticName(wanted)
	case sessions.SemanticMatchFuzzy:
		candidate = normalizeSemanticName(candidate)
		wanted = normalizeSemanticName(wanted)
		if candidate == "" || wanted == "" {
			return false
		}
		if candidate == wanted {
			return true
		}
		maxDistance := 1
		if len([]rune(wanted)) > 8 {
			maxDistance = 2
		}
		if len([]rune(wanted)) > 16 {
			maxDistance = 3
		}
		return semanticEditDistance(candidate, wanted, maxDistance) <= maxDistance
	default:
		return false
	}
}

func normalizeSemanticName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && out.Len() > 0 {
				out.WriteByte(' ')
			}
			out.WriteRune(r)
			space = false
			continue
		}
		space = out.Len() > 0
	}
	return strings.TrimSpace(out.String())
}

func semanticEditDistance(left, right string, limit int) int {
	a, b := []rune(left), []rune(right)
	if absInt(len(a)-len(b)) > limit {
		return limit + 1
	}
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range a {
		current[0] = i + 1
		rowMin := current[0]
		for j, rightRune := range b {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = minInt(current[j]+1, minInt(previous[j+1]+1, previous[j]+cost))
			if current[j+1] < rowMin {
				rowMin = current[j+1]
			}
		}
		if rowMin > limit {
			return limit + 1
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// Activation capture belongs to the native session backend and does not require
// semantic enumeration or reading accessibility text.
func (b *semanticBackend) CaptureActivation(ctx context.Context) (sessions.DesktopActivationContext, error) {
	observer, ok := b.pixels.(sessions.DesktopActivationObserver)
	if !ok {
		return sessions.DesktopActivationContext{}, sessions.ErrDesktopAdmission
	}
	return observer.CaptureActivation(ctx)
}

func (b *semanticBackend) VerifyWindowProcess(ctx context.Context, window uint64, pid uint32) error {
	verifier, ok := b.pixels.(sessions.DesktopWindowVerifier)
	if !ok {
		return sessions.ErrDesktopAdmission
	}
	return verifier.VerifyWindowProcess(ctx, window, pid)
}

func (b *semanticBackend) CaptureActivationImage(ctx context.Context) (sessions.DesktopActivationImage, error) {
	observer, ok := b.pixels.(sessions.DesktopActivationImageObserver)
	if !ok {
		return sessions.DesktopActivationImage{}, sessions.ErrDesktopAdmission
	}
	return observer.CaptureActivationImage(ctx)
}
