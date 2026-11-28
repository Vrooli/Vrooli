// Package experienceimport contains dry-run, reversible adapters from legacy
// experience-page sketches to the Workspace aggregate. It has no storage
// dependency: callers must inspect the receipt before deciding to persist.
package experienceimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	sketch "react-component-library/internal/sketch"
)

type sourcePage struct {
	Page struct {
		ID     string   `json:"id"`
		Title  string   `json:"title"`
		Routes []string `json:"routes"`
	} `json:"page"`
	Regions []struct {
		ID        string   `json:"id"`
		Purpose   string   `json:"purpose"`
		Elements  []string `json:"elements"`
		Lifecycle struct {
			Kind   string   `json:"kind"`
			States []string `json:"states"`
		} `json:"lifecycle"`
	} `json:"regions"`
	States []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
	} `json:"states"`
	Elements []struct {
		ID string `json:"id"`
	} `json:"elements"`
	Claims []struct {
		ID string `json:"id"`
	} `json:"claims"`
	Bindings struct {
		Elements map[string]json.RawMessage `json:"elements"`
		Regions  map[string]json.RawMessage `json:"regions"`
	} `json:"bindings"`
	Sketch json.RawMessage `json:"sketch"`
}

// PageContractEvidence inventories page obligations that a future native
// workspace renderer must preserve. It deliberately carries source IDs rather
// than translating them into a second contract language.
type PageContractEvidence struct {
	StateIDs          []string                  `json:"stateIds"`
	StateDescriptions []string                  `json:"stateDescriptions"`
	ElementIDs        []string                  `json:"elementIds"`
	ClaimIDs          []string                  `json:"claimIds"`
	RegionIDs         []string                  `json:"regionIds"`
	RegionLifecycles  []RegionLifecycleEvidence `json:"regionLifecycles"`
	ElementBindingIDs []string                  `json:"elementBindingIds"`
	RegionBindingIDs  []string                  `json:"regionBindingIds"`
}

type RegionLifecycleEvidence struct {
	RegionID string   `json:"regionId"`
	Kind     string   `json:"kind"`
	States   []string `json:"states"`
}

// InspectPageContract extracts obligations without claiming that the
// placement-only sketch can render or that the workspace can execute them.
func InspectPageContract(source []byte) (PageContractEvidence, error) {
	var page sourcePage
	if err := json.Unmarshal(source, &page); err != nil {
		return PageContractEvidence{}, fmt.Errorf("decode experience page contract: %w", err)
	}
	evidence := PageContractEvidence{}
	for _, state := range page.States {
		evidence.StateIDs = append(evidence.StateIDs, state.ID)
		evidence.StateDescriptions = append(evidence.StateDescriptions, state.Description)
	}
	for _, element := range page.Elements {
		evidence.ElementIDs = append(evidence.ElementIDs, element.ID)
	}
	for _, claim := range page.Claims {
		evidence.ClaimIDs = append(evidence.ClaimIDs, claim.ID)
	}
	for _, region := range page.Regions {
		evidence.RegionIDs = append(evidence.RegionIDs, region.ID)
		if region.Lifecycle.Kind != "" {
			evidence.RegionLifecycles = append(evidence.RegionLifecycles, RegionLifecycleEvidence{RegionID: region.ID, Kind: region.Lifecycle.Kind, States: append([]string(nil), region.Lifecycle.States...)})
		}
	}
	for id := range page.Bindings.Elements {
		evidence.ElementBindingIDs = append(evidence.ElementBindingIDs, id)
	}
	for id := range page.Bindings.Regions {
		evidence.RegionBindingIDs = append(evidence.RegionBindingIDs, id)
	}
	sort.Strings(evidence.ElementBindingIDs)
	sort.Strings(evidence.RegionBindingIDs)
	return evidence, nil
}

// Receipt is the reversible source record for a dry-run conversion. Both the
// whole authored page and its exact sketch member are retained byte-for-byte.
type Receipt struct {
	Version          int                  `json:"version"`
	Scenario         string               `json:"scenario"`
	Page             string               `json:"page"`
	WorkspaceID      string               `json:"workspaceId"`
	SourceHash       string               `json:"sourceHash"`
	SourceRevision   string               `json:"sourceRevision"`
	SourcePageJSON   []byte               `json:"sourcePageJson"`
	SourceSketchJSON []byte               `json:"sourceSketchJson"`
	TargetHash       string               `json:"targetHash"`
	Parity           bool                 `json:"parity"`
	RenderParity     bool                 `json:"renderParity"`
	ParityDetail     string               `json:"parityDetail"`
	RenderGap        string               `json:"renderGap,omitempty"`
	ContractGap      string               `json:"contractGap,omitempty"`
	RenderEvidence   RenderEvidence       `json:"renderEvidence"`
	ContractEvidence PageContractEvidence `json:"contractEvidence"`
}

// RenderEvidence describes the native subset of a conversion without
// suggesting that the legacy page is visually or behaviorally equivalent.
// Region IDs are stable source identities, so a persisted receipt can be
// audited without parsing the prose summary.
type RenderEvidence struct {
	MappedRegionIDs          []string               `json:"mappedRegionIds"`
	NativeButtonRegionIDs    []string               `json:"nativeButtonRegionIds"`
	LegacyPlacementRegionIDs []string               `json:"legacyPlacementRegionIds"`
	UnplacedItemCount        int                    `json:"unplacedItemCount"`
	FixtureCount             int                    `json:"fixtureCount"`
	RenderBindingCount       int                    `json:"renderBindingCount"`
	TemplateExport           string                 `json:"templateExport"`
	TemplateAsset            string                 `json:"templateAsset"`
	TemplateVersion          string                 `json:"templateVersion"`
	TemplatePorts            []TemplatePortEvidence `json:"templatePorts"`
}

// TemplatePortEvidence distinguishes a source port copied into editable page
// metadata from a port actually rendered by its native template shell. The
// current workspace adapter only does the former.
type TemplatePortEvidence struct {
	RegionID     string   `json:"regionId"`
	TemplatePort string   `json:"templatePort"`
	SlotPath     []string `json:"slotPath"`
	NativeShell  bool     `json:"nativeShell"`
}

type Result struct {
	Workspace *workspacev1.Workspace
	Receipt   Receipt
}

// RestorePage validates a receipt's source evidence and returns the exact
// experience-page bytes that were present during conversion. Receipts can be
// serialized and stored by a reviewed migration caller without losing the
// original formatting or unknown fields.
func RestorePage(receipt Receipt) ([]byte, error) {
	if receipt.Version != 1 || !safeIdentity(receipt.Scenario) || !safeIdentity(receipt.Page) {
		return nil, fmt.Errorf("unsupported or incomplete experience recovery receipt")
	}
	if err := verifySourcePage(receipt, receipt.SourcePageJSON); err != nil {
		return nil, err
	}
	return append([]byte(nil), receipt.SourcePageJSON...), nil
}

// VerifySource checks that the current source page still matches a conversion
// receipt. A mismatch means the dry-run is stale and must be repeated before
// any persistence operation.
func VerifySource(receipt Receipt, currentSource []byte) error {
	if receipt.Version != 1 || !safeIdentity(receipt.Scenario) || !safeIdentity(receipt.Page) {
		return fmt.Errorf("unsupported or incomplete experience recovery receipt")
	}
	return verifySourcePage(receipt, currentSource)
}

func verifySourcePage(receipt Receipt, source []byte) error {
	if len(source) == 0 || len(receipt.SourcePageJSON) == 0 || !bytes.Equal(source, receipt.SourcePageJSON) {
		return fmt.Errorf("source page differs from recovery receipt")
	}
	digest := sha256.Sum256(source)
	if hex.EncodeToString(digest[:]) != receipt.SourceHash {
		return fmt.Errorf("source page does not match recovery receipt hash")
	}
	revision, err := contentHash(source)
	if err != nil || revision != receipt.SourceRevision {
		return fmt.Errorf("source page does not match recovery receipt revision")
	}
	var envelope sourcePage
	if err := json.Unmarshal(source, &envelope); err != nil {
		return fmt.Errorf("decode recovery source page: %w", err)
	}
	if envelope.Page.ID != receipt.Page || !sameJSON(envelope.Sketch, receipt.SourceSketchJSON) {
		return fmt.Errorf("recovery source page does not match receipt identity and sketch")
	}
	return nil
}

// VerifyResult checks the dry-run's source recovery receipt against the target
// aggregate before a caller persists either one. It detects stale or modified
// source bytes, target bytes, and workspace bindings; it does not establish
// render parity or authorize source deletion.
func VerifyResult(result Result) error {
	r := result.Receipt
	if r.Version != 1 || result.Workspace == nil {
		return fmt.Errorf("unsupported or incomplete experience import result")
	}
	if !safeIdentity(r.Scenario) || !safeIdentity(r.Page) || r.WorkspaceID == "" {
		return fmt.Errorf("receipt identity is incomplete")
	}
	if result.Workspace.GetId() != r.WorkspaceID || result.Workspace.GetBindingScenario() != r.Scenario || result.Workspace.GetBindingPage() != r.Page {
		return fmt.Errorf("workspace binding does not match recovery receipt")
	}
	sourcePage, err := RestorePage(r)
	if err != nil {
		return err
	}
	expected, err := ConvertPage(r.Scenario, r.Page, sourcePage)
	if err != nil {
		return fmt.Errorf("recompute conversion evidence: %w", err)
	}
	// Recompute the complete receipt, not just its explanatory render summary.
	// This also binds identity, parity claims, source revision, and recovery
	// bytes to the conversion algorithm after a receipt has been serialized and
	// loaded again.
	if !reflect.DeepEqual(expected.Receipt, r) {
		return fmt.Errorf("conversion receipt does not match the source conversion")
	}
	canonicalWorkspace, err := json.Marshal(result.Workspace)
	if err != nil {
		return fmt.Errorf("encode workspace target: %w", err)
	}
	targetHash := sha256.Sum256(canonicalWorkspace)
	if hex.EncodeToString(targetHash[:]) != r.TargetHash {
		return fmt.Errorf("workspace target does not match recovery receipt hash")
	}
	if len(result.Workspace.GetPages()) == 0 {
		return fmt.Errorf("workspace target has no page")
	}
	for _, display := range result.Workspace.GetPages()[0].GetDisplays() {
		for _, state := range display.GetStates() {
			sketchJSON, err := RestoreSketch(state.GetArgsJson())
			if err != nil || !sameJSON(sketchJSON, r.SourceSketchJSON) {
				return fmt.Errorf("workspace state %q cannot recover the source sketch", state.GetId())
			}
			var adapter stateAdapter
			if err := json.Unmarshal([]byte(state.GetArgsJson()), &adapter); err != nil || !reflect.DeepEqual(adapter.SourcePageContract, r.ContractEvidence) {
				return fmt.Errorf("workspace state %q does not preserve source page obligations", state.GetId())
			}
		}
	}
	return nil
}

type templateNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Props    map[string]any `json:"props"`
	Children []templateNode `json:"children,omitempty"`
}

type stateAdapter struct {
	LegacyExperienceComposition legacyStateAdapter   `json:"legacyExperienceComposition"`
	SourcePageContract          PageContractEvidence `json:"sourcePageContract"`
}

type legacyStateAdapter struct {
	Kind           string          `json:"kind"`
	Version        int             `json:"version"`
	SourceRevision string          `json:"sourceRevision"`
	SourceSketch   json.RawMessage `json:"sourceSketch"`
}

// ConvertPage performs a read-only conversion of one composition-bearing
// experience page. IDs are deterministic so repeating a dry-run produces an
// identical target.
func ConvertPage(scenario, page string, source []byte) (Result, error) {
	if !safeIdentity(scenario) || !safeIdentity(page) {
		return Result{}, fmt.Errorf("scenario and page must be safe single-segment identities")
	}
	var envelope sourcePage
	if err := json.Unmarshal(source, &envelope); err != nil {
		return Result{}, fmt.Errorf("decode experience page: %w", err)
	}
	if envelope.Page.ID != page {
		return Result{}, fmt.Errorf("page identity mismatch: requested %q, source has %q", page, envelope.Page.ID)
	}
	if len(envelope.Sketch) == 0 || string(envelope.Sketch) == "null" {
		return Result{}, fmt.Errorf("experience page has no sketch object")
	}
	var doc sketch.Document
	if err := json.Unmarshal(envelope.Sketch, &doc); err != nil {
		return Result{}, fmt.Errorf("decode source sketch: %w", err)
	}
	sourceRevision, err := contentHash(source)
	if err != nil {
		return Result{}, fmt.Errorf("hash source page revision: %w", err)
	}
	if doc.Template == nil || doc.Render == nil {
		return Result{}, fmt.Errorf("page has no template/render composition to migrate; placement-only intent does not define native Workspace content or behavior")
	}
	if len(doc.Render.Regions) == 0 {
		return Result{}, fmt.Errorf("template composition has no render regions")
	}
	declared := make([]sketch.Region, 0, len(envelope.Regions))
	for _, region := range envelope.Regions {
		declared = append(declared, sketch.Region{ID: region.ID, Note: region.Purpose, Elements: region.Elements})
	}
	if _, err := sketch.PrepareRender(sketch.Snapshot{ContentHash: sourceRevision, Document: doc, DeclaredRegions: declared}, "Missing fixture", "Region failed"); err != nil {
		return Result{}, fmt.Errorf("source composition is not renderable: %w", err)
	}

	pageID := "pg_" + page
	children := make([]templateNode, 0, len(doc.Render.Regions))
	placements := make(map[string]sketch.Placement, len(doc.Placements))
	for _, placement := range doc.Placements {
		placements[placement.Region] = placement
	}
	evidence := RenderEvidence{
		MappedRegionIDs:          make([]string, 0, len(doc.Render.Regions)),
		NativeButtonRegionIDs:    make([]string, 0),
		LegacyPlacementRegionIDs: make([]string, 0),
		UnplacedItemCount:        len(doc.Unplaced),
		FixtureCount:             len(doc.Render.Fixtures),
		RenderBindingCount:       len(doc.Render.Bindings),
		TemplateExport:           doc.Render.TemplateExport,
		TemplateAsset:            doc.Template.Asset,
		TemplateVersion:          doc.Template.Version,
		TemplatePorts:            make([]TemplatePortEvidence, 0, len(doc.Render.Regions)),
	}
	for i, region := range doc.Render.Regions {
		if region.ID == "" {
			return Result{}, fmt.Errorf("render region %d has no identity", i)
		}
		propsMap := map[string]any{
			"label":                region.ID,
			"legacyTemplateRegion": region.TemplateRegion,
			"legacySlot":           strings.Join(region.Slot, "."),
			"legacyParent":         region.Parent,
			"legacyExport":         region.Export,
			"legacyStory":          region.Story,
			"optional":             region.Optional,
		}
		slot := templateNode{ID: safeNodeID(region.ID, i), Type: "slot", Props: propsMap}
		evidence.MappedRegionIDs = append(evidence.MappedRegionIDs, region.ID)
		evidence.TemplatePorts = append(evidence.TemplatePorts, TemplatePortEvidence{
			RegionID: region.ID, TemplatePort: region.TemplateRegion,
			SlotPath: append([]string(nil), region.Slot...), NativeShell: false,
		})
		button, hasButton := nativeButtonFallback(region, placements[region.ID], doc.Render.Bindings[region.ID])
		if hasButton {
			slot.Children = []templateNode{button}
			evidence.NativeButtonRegionIDs = append(evidence.NativeButtonRegionIDs, region.ID)
		} else if _, placed := placements[region.ID]; placed {
			evidence.LegacyPlacementRegionIDs = append(evidence.LegacyPlacementRegionIDs, region.ID)
		}
		children = append(children, slot)
	}
	root := templateNode{ID: pageID + "_root", Type: "page", Props: map[string]any{"title": envelope.Page.Title}, Children: children}
	specJSON, err := json.Marshal(root)
	if err != nil {
		return Result{}, err
	}
	views := doc.Intent
	viewports := []string{"desktop"}
	if views != nil && len(views.Viewports) > 0 {
		viewports = dedupe(viewsFromIntent(views.Viewports))
	}
	contractEvidence, err := InspectPageContract(source)
	if err != nil {
		return Result{}, fmt.Errorf("inspect source page obligations: %w", err)
	}
	statePayload, err := json.Marshal(stateAdapter{LegacyExperienceComposition: legacyStateAdapter{Kind: "legacy-render-adapter", Version: 1, SourceRevision: sourceRevision, SourceSketch: envelope.Sketch}, SourcePageContract: contractEvidence})
	if err != nil {
		return Result{}, err
	}
	pageType := doc.Template.Asset
	workspace := &workspacev1.Workspace{
		Id:              "ws_experience_" + scenario + "_" + page,
		Name:            envelope.Page.Title,
		Summary:         "Imported from scenario experience intent; legacy render payload is retained in the state adapter.",
		Provenance:      workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED,
		BindingScenario: scenario,
		BindingPage:     page,
		ThemeJson:       "{}",
		OptionsJson:     "{}",
		Revision:        1,
		Pages: []*workspacev1.WorkspacePage{{
			Id: pageID, Name: envelope.Page.Title, Route: first(envelope.Page.Routes), PageTypeId: pageType, SpecJson: string(specJSON),
			Displays: makeDisplays(page, viewports, string(statePayload)),
		}},
	}
	sourceHash := sha256.Sum256(source)
	canonicalWorkspace, err := json.Marshal(workspace)
	if err != nil {
		return Result{}, err
	}
	targetHash := sha256.Sum256(canonicalWorkspace)
	// The lossless adapter is the parity boundary. Native nodes expose every
	// mapped render port, while the untouched sketch payload remains recoverable.
	restored, err := RestoreSketch(workspace.Pages[0].Displays[0].States[0].ArgsJson)
	if err != nil || !sameJSON(restored, envelope.Sketch) {
		return Result{}, fmt.Errorf("conversion parity failed: adapter did not preserve source sketch")
	}
	return Result{Workspace: workspace, Receipt: Receipt{
		Version: 1, Scenario: scenario, Page: page, WorkspaceID: workspace.Id,
		SourceHash: hex.EncodeToString(sourceHash[:]), SourceRevision: sourceRevision, SourcePageJSON: append([]byte(nil), source...), SourceSketchJSON: append([]byte(nil), envelope.Sketch...),
		TargetHash: hex.EncodeToString(targetHash[:]), Parity: true, RenderParity: false,
		ParityDetail:   fmt.Sprintf("the exact source page and sketch are recoverable; %d of %d render regions map to native slot nodes and %d static Button placements map to native button fallbacks", len(evidence.MappedRegionIDs), len(doc.Render.Regions), len(evidence.NativeButtonRegionIDs)),
		RenderGap:      renderGap(doc, evidence),
		ContractGap:    contractGap(contractEvidence),
		RenderEvidence: evidence, ContractEvidence: contractEvidence,
	}}, nil
}

func contractGap(evidence PageContractEvidence) string {
	return fmt.Sprintf("%d source states, %d elements, %d claims, and %d region lifecycles are retained as page-contract evidence; native workspace behavior and claim evaluation are not implemented", len(evidence.StateIDs), len(evidence.ElementIDs), len(evidence.ClaimIDs), len(evidence.RegionLifecycles))
}

func renderGap(doc sketch.Document, evidence RenderEvidence) string {
	ports := make([]string, 0, len(evidence.TemplatePorts))
	for _, port := range evidence.TemplatePorts {
		name := port.TemplatePort
		if name == "" {
			name = port.RegionID
		}
		ports = append(ports, name)
	}
	return fmt.Sprintf("native workspace stores %d region nodes and %d static Button fallbacks, but does not render the %s@%s shell or its ports [%s]; %d other placed assets, %d unplaced items, %d render bindings, and %d fixtures remain owned by the legacy renderer", len(evidence.MappedRegionIDs), len(evidence.NativeButtonRegionIDs), doc.Template.Asset, doc.Template.Version, strings.Join(ports, ", "), len(evidence.LegacyPlacementRegionIDs), evidence.UnplacedItemCount, evidence.RenderBindingCount, evidence.FixtureCount)
}

// nativeButtonFallback covers one stable, declarative part of legacy
// compositions: a placed shared Button with a static authored label. It is a
// visual fallback inside the native slot; behavior remains owned by the legacy
// adapter until its interaction bindings have native equivalents.
func nativeButtonFallback(region sketch.RenderRegion, placement sketch.Placement, binding any) (templateNode, bool) {
	if placement.Fills.Asset != "controls.button" || region.Export != "Button" {
		return templateNode{}, false
	}
	values, ok := binding.(map[string]any)
	if !ok {
		return templateNode{}, false
	}
	label, ok := values["children"].(string)
	if !ok || strings.TrimSpace(label) == "" {
		return templateNode{}, false
	}
	return templateNode{
		ID:   safeNodeID(region.ID, 0) + "-button",
		Type: "button",
		Props: map[string]any{
			"text":      label,
			"variant":   "primary",
			"shape":     "pill",
			"ariaLabel": label,
		},
	}, true
}

func contentHash(source []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// RestoreSketch returns the exact legacy composition object held by the state
// adapter. It is intentionally a rollback aid, not a renderer implementation.
func RestoreSketch(argsJSON string) (json.RawMessage, error) {
	var adapter stateAdapter
	if err := json.Unmarshal([]byte(argsJSON), &adapter); err != nil {
		return nil, err
	}
	if adapter.LegacyExperienceComposition.Kind != "legacy-render-adapter" || adapter.LegacyExperienceComposition.Version != 1 || len(adapter.LegacyExperienceComposition.SourceSketch) == 0 {
		return nil, fmt.Errorf("workspace state does not contain a legacy experience adapter")
	}
	return adapter.LegacyExperienceComposition.SourceSketch, nil
}

func makeDisplays(page string, viewports []string, args string) []*workspacev1.WorkspaceDisplay {
	out := make([]*workspacev1.WorkspaceDisplay, 0, len(viewports))
	for i, viewport := range viewports {
		device, width, height := viewportSize(viewport)
		out = append(out, &workspacev1.WorkspaceDisplay{Id: fmt.Sprintf("dp_%s_%s", page, viewport), Device: device, Width: width, Height: height, SortOrder: int32(i), States: []*workspacev1.WorkspaceState{{Id: fmt.Sprintf("st_%s_%s_default", page, viewport), Name: "Default", ArgsJson: args, Status: workspacev1.StateStatus_STATE_STATUS_DRAFT}}})
	}
	return out
}

func viewportSize(value string) (workspacev1.DisplayDevice, int32, int32) {
	switch strings.ToLower(value) {
	case "phone":
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE, 390, 844
	case "tablet":
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET, 768, 1024
	default:
		return workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP, 1440, 900
	}
}

func viewsFromIntent(values []string) []string {
	var out []string
	for _, value := range values {
		switch strings.ToLower(value) {
		case "phone", "tablet", "desktop":
			out = append(out, strings.ToLower(value))
		}
	}
	return out
}

func safeNodeID(value string, index int) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return fmt.Sprintf("slot-%d", index)
	}
	return b.String()
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return []string{"desktop"}
	}
	return out
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func safeIdentity(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			if index == 0 && (r == '_' || r == '-') {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func sameJSON(a, b []byte) bool {
	var left, right any
	return json.Unmarshal(a, &left) == nil && json.Unmarshal(b, &right) == nil && equalJSON(left, right)
}

func equalJSON(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}
