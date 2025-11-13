# BAS Node Implementation Plan

**Version:** 1.0
**Created:** 2025-11-10
**Status:** In Progress
**Owner:** BAS Team

---

## 📋 Executive Summary

This document outlines the implementation roadmap for expanding BAS from 9 nodes to 27+ nodes, organized into 5 phases over approximately 4-6 weeks. The plan prioritizes unblocking testing infrastructure, then enabling common workflows, followed by advanced control flow and mobile/context features.

**Current State (2025-11-13):** 33 compiler/runtime/UI nodes implemented across 7 categories with `go test ./... -cover` succeeding; targeted Vitest palette suite passes while the full run still OOMs under tinypool (see Latest Validation); Browserless integration playbooks remain blocked on fixture availability.
**Phase Completion:** Phases 1–6 have shipped (see tracker below); remaining effort is focused on integration playbooks, requirement coverage, and perf/UX validation.
**Final Goal:** 27 nodes, 100% test coverage, organized UI — ✅ Node target exceeded (33 live)

---

## 🎯 Goals & Success Metrics

### Primary Goals
1. **Unblock Testing**: Fix integration test failures (2 unsupported node types)
2. **Enable Common Workflows**: Cover 85%+ of real-world automation scenarios
3. **Improve UX**: Organize nodes into searchable, collapsible categories
4. **Enable Data-Driven Testing**: Variables + control flow for reusable workflows

### Success Metrics
- **Test Coverage**: 100% integration tests passing *(current: `go test ./... -cover` + Vitest pass; integration playbooks pending)*
- **Node Coverage**: 27 nodes across 7 categories *(current: 33 nodes live — goal exceeded)*
- **Workflow Complexity**: Support conditional logic, loops, variables *(met — conditional/loop/variable nodes operational with unit coverage)*
- **Mobile Support**: Gesture, rotation, viewport testing *(met — gesture + rotate nodes live; awaiting dedicated mobile fixture validation)*
- **API Testing**: Network mocking, cookie/storage manipulation *(met — nodes + docs shipped)*
- **Time to Find Node**: <5 seconds for new users *(search + quick access shipped; formal usability test outstanding)*
- **Workflow Reusability**: Variables enable DRY principles *(met — set/use/workflow-call nodes + new variable tests)*

---

## 🗺️ Implementation Phases

### Phase 1: Unblock Testing (1-2 days) ⚡
**Priority:** CRITICAL
**Goal:** Fix failing integration tests

#### 1.1 Implement Script/Evaluate Node
**Status:** 🟢 Completed — backend/runtime + UI shipped (2025-11-15)
**Assignee:** BAS Team
**Effort:** 4-6 hours
**Blocking:** `test/playbooks/replay/render-check.json`

**Files to Modify:**
- `api/browserless/compiler/compiler.go`: Add `StepEvaluate` constant
- `api/browserless/runtime/instructions.go`: Add evaluate case to `instructionFromStep()`
- `api/browserless/cdp/adapter.go`: Already has case at line 98, verify implementation
- `ui/src/components/NodePalette.tsx`: Add Script node definition
- `ui/src/components/nodes/ScriptNode.tsx`: Create node component

**Implementation Checklist:**
- [x] Add `StepEvaluate StepType = "evaluate"` to compiler constants
- [x] Add evaluate to `supportedStepTypes` map
- [x] Implement `evaluateConfig` struct in `runtime/instructions.go`:
  ```go
  type evaluateConfig struct {
      Expression string `json:"expression"`
      TimeoutMs  int    `json:"timeoutMs"`
      StoreResult string `json:"storeResult,omitempty"` // Future: variable name
  }
  ```
- [x] Add case in `instructionFromStep()` to handle evaluate
- [x] Verify CDP adapter's `ExecuteEvaluate()` handles timeout correctly
- [x] Add Go tests: `TestEvaluateNode_*` in `runtime/instructions_test.go`
- [x] Add UI component with Monaco editor for script input
- [x] Add requirement tag: `[REQ:BAS-NODE-SCRIPT-EXECUTE]`
- [x] Update `docs/requirements-traceability.md`
- [ ] Test with `render-check.json` workflow (still pending end-to-end run)

**Acceptance Criteria:**
- `evaluate` node compiles successfully ✅ (`go test ./browserless/runtime`)
- CDP executes JavaScript and returns result ✅ (`ExecuteEvaluate` wired via adapter)
- Timeout handling works (30s default) ✅ (runtime + adapter defaults)
- Workflow `replay/render-check.json` passes ⚠️ pending playbook run once CI resources available
- Unit tests cover success, error, timeout cases ✅ (`runtime/instructions_test.go`)

---

#### 1.2 Implement Keyboard Node (or Document Shortcut)
**Status:** 🟢 Completed — raw key dispatch now live (2025-11-15)
**Assignee:** BAS Team
**Effort:** 3-4 hours
**Blocking:** `test/playbooks/ui/projects/new-project-validation.json`

**Decision Point:** Separate node vs. extending Shortcut
- **Recommendation:** Separate node for granular control
- **Shortcut:** High-level combos (Ctrl+C, Cmd+V)
- **Keyboard:** Low-level events (keydown/keyup/keypress, press-hold)

**Files to Modify:**
- `api/browserless/compiler/compiler.go`: Add `StepKeyboard` constant
- `api/browserless/runtime/instructions.go`: Add keyboard case
- `api/browserless/cdp/actions.go`: Add `ExecuteKeyboard()` method
- `ui/src/components/NodePalette.tsx`: Add Keyboard node

**Implementation Checklist:**
- [x] Add `StepKeyboard StepType = "keyboard"` to compiler
- [x] Implement `keyboardConfig` struct:
  ```go
  type keyboardConfig struct {
      Key       string `json:"key"`       // Key name (e.g., "Enter", "a", "ArrowDown")
      EventType string `json:"eventType"` // "keydown", "keyup", "keypress"
      Modifiers struct {
          Ctrl  bool `json:"ctrl"`
          Shift bool `json:"shift"`
          Alt   bool `json:"alt"`
          Meta  bool `json:"meta"`
      } `json:"modifiers"`
      DelayMs   int `json:"delayMs"`
      TimeoutMs int `json:"timeoutMs"`
  }
  ```
- [x] Implement CDP keyboard dispatch (CDP `Input.dispatchKeyEvent`)
- [x] Add Go tests for all event types
- [x] Add UI component with key picker
- [ ] Test with validation workflow (playbook still queued)
- [x] Document difference from Shortcut in README

**Acceptance Criteria:**
- Keyboard events dispatch correctly (keydown, keyup, keypress) ✅ (`ExecuteKeyboard` end-to-end)
- Modifiers work (Ctrl, Shift, Alt, Meta) ✅ (unit coverage + CDP wiring)
- Press-hold simulation works (keydown → delay → keyup) ✅ (delay support in runtime + CDP)
- Workflow using keyboard node passes ⚠️ validation playbook pending queued CI slot

---

#### 1.3 Fix Navigate "invalid context" Errors
**Status:** 🟢 Completed — scenario registry + resolver shipped (2025-11-15)
**Assignee:** BAS Team
**Effort:** 2-3 hours
**Blocking:** 8/11 integration tests

**Root Cause Analysis:**
- Workflows with `destinationType: "scenario"` fail at first navigate step
- Error: "invalid context" suggests URL resolution failing
- Likely issue: scenario name → URL conversion broken

**Files to Investigate:**
- `api/browserless/runtime/instructions.go`: Line ~100-150 (navigate case)
- `internal/scenarioport/resolver.go`: Scenario port/URL resolution
- `test/playbooks/ui/projects/new-project-create.json`: Example workflow

- [x] Debug scenario URL resolution in `instructionFromStep()`
- [x] Verify `scenarioport.ResolveScenarioURL()` works correctly
- [x] Check if `SCENARIO_REGISTRY` env var is set during tests (documented + env override support)
- [x] Add logging for resolved URLs in navigate instruction
- [ ] Test manually: `./cli/browser-automation-studio execution adhoc --workflow test/playbooks/ui/projects/new-project-dialog-open.json --verbose`
- [x] Fix URL resolution logic if broken
- [x] Add unit tests for scenario URL resolution
- [ ] Re-run integration phase: `./test/run-tests.sh integration`

- Navigate with `destinationType: "scenario"` resolves URL correctly ✅ (`scenarioport.ResolveURL` + runtime tests)
- Integration tests pass: 9/11 (2 empty JSONs remain expected to fail) ⚠️ waiting on full integration sweep
- Verbose logging shows resolved URL for debugging ✅ (`instructionFromStep` log line)

---

### Phase 2: Core UX Nodes (3-5 days) 🎯
**Priority:** HIGH
**Goal:** Cover 85% of common workflows

#### 2.1 Implement Hover Node
**Status:** 🟢 Completed — smooth cursor hover shipped (2025-11-15)
**Assignee:** BAS Team
**Effort:** 3-4 hours
**Impact:** Critical for dropdown menus, tooltips, mega-menus

**Files to Create/Modify:**
- `api/browserless/compiler/compiler.go`: Add `StepHover`
- `api/browserless/runtime/instructions.go`: Add hover case
- `api/browserless/cdp/actions.go`: Add `ExecuteHover()` method
- `ui/src/components/NodePalette.tsx`: Add Hover node
- `ui/src/components/nodes/HoverNode.tsx`: Create component

**Implementation Checklist:**
- [x] Add `StepHover StepType = "hover"` to compiler
- [x] Implement `hoverConfig` struct:
  ```go
  type hoverConfig struct {
      Selector   string `json:"selector"`
      TimeoutMs  int    `json:"timeoutMs"`
      WaitForMs  int    `json:"waitForMs"`
      Steps      int    `json:"steps"`      // Number of intermediate points for smooth movement
      DurationMs int    `json:"durationMs"` // Total hover duration
  }
  ```
- [x] Implement CDP hover using `Input.dispatchMouseEvent`:
  ```go
  // Pseudo-code
  element := findElement(selector)
  box := getBoundingBox(element)
  centerX, centerY := box.center()
-  dispatchMouseEvent(type="mouseMoved", x=centerX, y=centerY)
-  wait(durationMs)
  ```
- [x] Add smooth movement (multiple mouseMoved events for natural motion)
- [x] Add Go tests: hover success, element not found, timeout
- [x] Add UI component with selector + duration inputs
- [x] Add requirement: `[REQ:BAS-NODE-HOVER-INTERACTION]`
- [ ] Create test workflow: hover menu → assert submenu visible

**Acceptance Criteria:**
- Hover triggers `:hover` CSS states ✅ (CDP pointer glide with `dom.GetBoxModel` centering)
- Dropdowns open on hover ⚠️ pending workflow-level validation once hover playbook lands
- Tooltips appear on hover ⚠️ pending same workflow validation
- Smooth mouse movement option works ✅ (configurable steps/duration)
- Tests cover edge cases (element moves, disappears) ✅ (`instructions_test.go` + CDP pointer helper); scenario playbook still pending

---

#### 2.2 Implement Scroll Node
**Status:** 🟢 Completed — page/element scrolling + visibility sweeps shipped (2025-11-16)
**Assignee:** TBD
**Effort:** 4-5 hours
**Impact:** Essential for lazy-load, infinite scroll, below-fold elements

**Files to Create/Modify:**
- `api/browserless/compiler/compiler.go`: Add `StepScroll`
- `api/browserless/runtime/instructions.go`: Add scroll case
- `api/browserless/cdp/actions.go`: Add `ExecuteScroll()` method
- `ui/src/components/nodes/ScrollNode.tsx`: Create component

**Implementation Checklist:**
- [x] Add `StepScroll StepType = "scroll"` to compiler
- [x] Implement `scrollConfig` struct:
  ```go
  type scrollConfig struct {
      ScrollType string `json:"scrollType"` // "page", "element", "position"
      Selector   string `json:"selector"`   // For element scroll
      Direction  string `json:"direction"`  // "up", "down", "left", "right", "top", "bottom"
      Amount     int    `json:"amount"`     // Pixels or percentage
      X          int    `json:"x"`          // For position scroll
      Y          int    `json:"y"`          // For position scroll
      Behavior   string `json:"behavior"`   // "auto", "smooth"
      TimeoutMs  int    `json:"timeoutMs"`
      WaitForMs  int    `json:"waitForMs"`
  }
  ```
- [x] Implement scroll behaviors:
  - **Page scroll**: `window.scrollBy()` or `window.scrollTo()`
  - **Element scroll**: `element.scrollIntoView({behavior, block, inline})`
  - **Position scroll**: `window.scrollTo(x, y)`
- [x] Add smooth scrolling option
- [x] Add "scroll until element visible" variant
- [x] Add Go tests for all scroll types
- [x] Add UI component with scroll type picker + options
- [x] Add requirement: `[REQ:BAS-NODE-SCROLL-NAVIGATION]`
- [ ] Test with lazy-load scenario

**Acceptance Criteria:**
- Page scrolls to top/bottom/position correctly
- Element scrollIntoView works with smooth behavior
- Lazy-loaded content triggers on scroll
- Tests cover all scroll types and directions

---

#### 2.3 Implement Select Node (Dropdown)
**Status:** 🟢 Completed — native + multi-select support merged (2025-11-16)
**Assignee:** BAS Team
**Effort:** 3-4 hours
**Impact:** Forms, filters, configuration UIs

**Files to Create/Modify:**
- `api/browserless/compiler/compiler.go`: Add `StepSelect`
- `api/browserless/runtime/instructions.go`: Add select case
- `api/browserless/cdp/actions.go`: Add `ExecuteSelect()` method
- `ui/src/components/nodes/SelectNode.tsx`: Create component

**Implementation Checklist:**
- [x] Add `StepSelect StepType = "select"` to compiler
- [x] Implement `selectConfig` struct:
  ```go
  type selectConfig struct {
      Selector   string   `json:"selector"`   // <select> element selector
      SelectBy   string   `json:"selectBy"`   // "value", "text", "index"
      Value      string   `json:"value"`      // Option value to select
      Text       string   `json:"text"`       // Option text to select (fuzzy match)
      Index      int      `json:"index"`      // Option index (0-based)
      Multiple   bool     `json:"multiple"`   // Multi-select support
      Values     []string `json:"values"`     // For multi-select
      TimeoutMs  int      `json:"timeoutMs"`
      WaitForMs  int      `json:"waitForMs"`
  }
  ```
- [x] Implement selection methods:
  ```javascript
  // By value
  select.value = 'option-value';

  // By text (fuzzy match)
  Array.from(select.options).find(opt => opt.text.includes(text)).selected = true;

  // By index
  select.selectedIndex = index;

  // Multi-select
  Array.from(select.options).forEach((opt, i) => {
      opt.selected = values.includes(opt.value);
  });

  // Trigger change event
  select.dispatchEvent(new Event('change', {bubbles: true}));
  ```
- [x] Add Go tests for all selection methods (runtime normalization + validation)
- [x] Add UI component with selection method picker
- [x] Add requirement: `[REQ:BAS-NODE-SELECT-DROPDOWN]`
- [ ] Test with form submission workflow

**Acceptance Criteria:**
- Select by value/text/index works ✅ (runtime validation + CDP script dispatching input/change events)
- Multi-select works ✅ (CDP enforces `<select multiple>` and syncs selected options)
- Change events fire correctly ✅ (input + change fired after script execution)
- Works with custom dropdowns (React Select, etc.) via text matching ⚠️ limited to native/select-with-option structures; bespoke widgets still require future work
- Tests cover edge cases (option not found, disabled options) ✅ runtime tests assert validation for missing values, negative indices, and multi-select data

---

#### 2.4 Implement Variable System (Set + Use)
**Status:** 🟡 In Progress — runtime + UI nodes landed (2025-11-16); end-to-end workflow + variable picker polish still open
**Assignee:** TBD
**Effort:** 8-12 hours (COMPLEX - requires execution context threading)
**Impact:** Transforms BAS from record-replay to programmable automation

**Architecture Decision:**
- **Scope:** Workflow-scoped only (no global/project variables for now)
- **Storage:** In-memory context map during execution
- **Persistence:** Variables saved in execution artifacts for debugging
- **Types:** String, number, boolean, JSON object/array

**Files to Create/Modify:**
- `api/browserless/compiler/compiler.go`: Add `StepSetVariable`, `StepUseVariable`
- `api/browserless/runtime/context.go`: Create new file for execution context
- `api/browserless/runtime/instructions.go`: Add variable cases
- `api/browserless/client.go`: Thread context through execution pipeline
- `ui/src/components/nodes/SetVariableNode.tsx`: Create component
- `ui/src/components/nodes/UseVariableNode.tsx`: Create component

**Implementation Checklist:**

**Phase 2.4.1: Core Infrastructure (4-5 hours)**
- [x] Create `runtime/context.go`:
  ```go
  type ExecutionContext struct {
      Variables map[string]interface{} `json:"variables"`
      mu        sync.RWMutex
  }

  func (ctx *ExecutionContext) Set(key string, value interface{}) error
  func (ctx *ExecutionContext) Get(key string) (interface{}, bool)
  func (ctx *ExecutionContext) GetString(key string) (string, error)
  func (ctx *ExecutionContext) GetInt(key string) (int, error)
  func (ctx *ExecutionContext) GetBool(key string) (bool, error)
  func (ctx *ExecutionContext) Delete(key string)
  func (ctx *ExecutionContext) Has(key string) bool
  func (ctx *ExecutionContext) Snapshot() map[string]interface{}
  ```
- [x] Update `Instruction` struct to include context
- [x] Thread context through `Client.ExecuteWorkflow()` → `session.ExecuteInstruction()`
- [x] Persist context snapshot in execution artifacts (for debugging)

**Phase 2.4.2: Set Variable Node (2-3 hours)**
- [x] Add `StepSetVariable StepType = "setVariable"` to compiler
- [x] Implement `setVariableConfig` struct:
  ```go
  type setVariableConfig struct {
      Name       string      `json:"name"`       // Variable name
      Value      interface{} `json:"value"`      // Static value
      SourceType string      `json:"sourceType"` // "static", "extract", "expression"
      Selector   string      `json:"selector"`   // For extract source
      Expression string      `json:"expression"` // For JS expression evaluation
      ExtractType string     `json:"extractType"` // "text", "attribute", etc.
  }
  ```
- [x] Implement variable assignment logic
- [x] Support extraction into variable (Extract node output → variable)
- [x] Add Go tests for all source types
- [x] Add UI component with source type picker
- [x] Add requirement: `[REQ:BAS-NODE-VARIABLE-SET]`

**Phase 2.4.3: Use Variable Node (2-3 hours)**
- [x] Variable interpolation in string params: `Hello {{username}}!`
- [x] Update all node param parsing to support `{{varName}}` syntax
- [x] Implement template replacement in `instructionFromStep()`:
  ```go
  func interpolateVariables(template string, ctx *ExecutionContext) string {
      re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
      return re.ReplaceAllStringFunc(template, func(match string) string {
          varName := strings.Trim(match, "{} ")
          if val, ok := ctx.Get(varName); ok {
              return fmt.Sprintf("%v", val)
          }
          return match // Keep original if var not found
      })
  }
  ```
- [x] Add validation: warn if variable used but not set
- [x] Add Go tests for interpolation in all node types
- [x] Update UI to show variable picker/autocomplete (Set/Use nodes expose datalist suggestions via `useWorkflowVariables`)
- [x] Add requirement: `[REQ:BAS-NODE-VARIABLE-USE]`

**Phase 2.4.4: Integration & Testing (1-2 hours)**
- [x] Create end-to-end test workflow (navigate → extract → setVariable → assert use)
  ```json
  {
    "nodes": [
      {"type": "navigate", "data": {"url": "https://example.com"}},
      {"type": "extract", "data": {"selector": "h1", "storeIn": "pageTitle"}},
      {"type": "setVariable", "data": {"name": "greeting", "value": "Hello {{pageTitle}}!"}},
      {"type": "assert", "data": {"expression": "greeting.includes('Example')"}}
    ]
  }
  ```
- [x] Verify variable snapshots stored in execution artifacts (manual run against local scenario)
- [x] Test variable interpolation coverage outside unit tests (click/type/scroll params)
- [x] Update documentation with variable examples + UI walkthrough

_Notes:_ Covered by `TestVariableFlowEndToEnd` in `api/browserless/client_test.go`, plus README/UI updates describing the picker and workflow pattern.

**Acceptance Criteria:**
- Variables set and retrieved correctly
- Extraction results stored in variables
- Variables interpolated in all string parameters
- Context persisted in execution artifacts for debugging
- Tests cover edge cases (undefined variables, type mismatches)
- Documentation shows common patterns (login credentials, form data)

---

### Phase 3: Advanced Interaction (3-5 days) 💪
**Priority:** MEDIUM
**Goal:** Enable complex workflows (auth, multi-window, file uploads)

#### 3.1 Implement Focus/Blur Nodes
**Status:** 🟢 Completed — runtime + UI shipped 2025-11-16
**Effort:** 2-3 hours (actual ~3.5h including UI polish + tests)
**Impact:** Form validation, accessibility testing

**Implementation Checklist:**
- [x] Add `StepFocus`, `StepBlur` to compiler
- [x] Implement CDP focus/blur helpers (`chromedp.Focus` + blur script)
- [x] Add validation + timing tests in `runtime/instructions_test.go`
- [x] Create UI components with element picker + timing inputs (`FocusNode`, `BlurNode`)
- [x] Add tests and requirement tags `[REQ:BAS-NODE-FOCUS-INPUT]`, `[REQ:BAS-NODE-BLUR-VALIDATION]`
- [x] Document use cases + requirements entries (plan + `requirements-traceability.md`)

**Notes:** Vitest suite updated to expect the larger palette. Focus node defaults to 5s timeout, Blur defaults to 150ms wait to catch validation popups.

---

#### 3.2 Implement Drag & Drop Node
**Status:** 🟢 Completed — HTML5 drag-drop + UI shipped (2025-11-17)
**Effort:** 6-8 hours (COMPLEX - HTML5 drag API simulation)
**Impact:** Kanban boards, file uploads, sortable lists

**Implementation Checklist:**
- [x] Add `StepDragDrop` to compiler
- [x] Implement `dragDropConfig` struct:
  ```go
  type dragDropConfig struct {
      SourceSelector string `json:"sourceSelector"`
      TargetSelector string `json:"targetSelector"`
      HoldMs         int    `json:"holdMs"`
      Steps          int    `json:"steps"`
      DurationMs     int    `json:"durationMs"`
      OffsetX        int    `json:"offsetX"`
      OffsetY        int    `json:"offsetY"`
      TimeoutMs      int    `json:"timeoutMs"`
      WaitForMs      int    `json:"waitForMs"`
  }
  ```
- [x] Implement HTML5 drag-drop event simulation (DataTransfer polyfill + DragEvent dispatch)
- [x] Add CDP pointer simulation fallback (Input.dispatchMouseEvent w/ smooth cursor path)
- [x] Add requirement: `[REQ:BAS-NODE-DRAG-DROP]`
- [ ] Test with Trello-like board (pending scenario fixture; covered by unit + vitest suites for now)

**Notes:**
- Runtime clamps hold/step/duration/offset ranges and persists offsets for debugging.
- CDP session now emits real pointer movement plus synthetic DragEvent sequence so React/HTML5 boards receive `dragenter/dragover/drop` reliably.
- New React Flow node (`DragDropNode`) exposes selectors, hold delay, smoothness, offsets, and wait timing w/ contextual hints; palette + WorkflowBuilder register the node.
- Manual Trello-style scenario still queued while we stabilise scenario launcher capacity.

---

#### 3.3 Implement Tab/Window Switch Node
**Status:** 🟢 Completed — CDP target registry + UI shipped (2025-11-18)
**Effort:** 5-7 hours (COMPLEX - CDP Target management)
**Impact:** Multi-window workflows, OAuth popups, "open in new tab"

**Implementation Checklist:**
- [x] Add `StepTabSwitch` to compiler
- [x] Implement `tabSwitchConfig` struct:
  ```go
  type tabSwitchConfig struct {
      SwitchBy   string `json:"switchBy"`   // "index", "title", "url", "newest", "oldest"
      Index      int    `json:"index"`      // Tab index (0-based)
      TitleMatch string `json:"titleMatch"` // Regex or substring
      URLMatch   string `json:"urlMatch"`   // Regex or substring
      WaitForNew bool   `json:"waitForNew"` // Wait for new tab to open
      TimeoutMs  int    `json:"timeoutMs"`
      CloseOld   bool   `json:"closeOld"`   // Close previous tab after switch
  }
  ```
- [x] Update `cdp/session.go` to track multiple targets
- [x] Implement CDP `Target.setDiscoverTargets()` and `Target.attachToTarget()`
- [x] Add tab/window inventory management with wait-for-new polling + cleanup
- [ ] Test OAuth popup flow (still pending scenario fixture)
- [x] Add requirement: `[REQ:BAS-NODE-TAB-SWITCH]`

**Notes:**
- CDP target inventory now filters for `page` types, records creation order, and exposes helper APIs for selection + waits.
- Browser-level listeners keep the registry accurate even when Headless Chrome spawns popups outside the main target.
- `closeOld` option calls `Target.closeTarget` so workflows don’t leak tabs when hopping through OAuth.
- Manual OAuth fixture still pending to exercise the wait-for-new branch end-to-end.

---

#### 3.4 Implement Upload File Node
**Status:** 🟢 Completed — Browser + UI wiring merged (2025-11-18)
**Effort:** 3-4 hours
**Impact:** Form testing, attachment uploads

**Implementation Checklist:**
- [x] Add `StepUploadFile` to compiler
- [x] Implement `uploadFileConfig` struct:
  ```go
  type uploadFileConfig struct {
      Selector  string   `json:"selector"`  // <input type="file"> selector
      FilePath  string   `json:"filePath"`  // Absolute path to file
      FilePaths []string `json:"filePaths"` // For multiple files
      TimeoutMs int      `json:"timeoutMs"`
      WaitForMs int      `json:"waitForMs"`
  }
  ```
- [x] Implement CDP `DOM.setFileInputFiles()` method
- [x] Support single and multiple file uploads
- [x] Add file existence validation
- [ ] Test with image/document upload form
- [x] Add requirement: `[REQ:BAS-NODE-UPLOAD-FILE]`

**Notes:**
- File paths must be absolute on execution machine
- Consider future: base64 inline file content for portability
- Handle drag-drop file uploads (redirect to drag-drop node)

#### 3.5 Implement Workflow Call Node
**Status:** 🟢 Completed — inline orchestration shipped (2025-11-23)
**Effort:** 6-7 hours (COMPLEX — recursive execution + context hygiene)
**Impact:** Workflow reuse, shared auth/login helpers, modular automation libraries

**Implementation Checklist:**
- [x] Promote `workflowCall` from compiler-only to runtime-supported (workflowId required, async blocked for now)
- [x] Fetch referenced workflows from the repository, compile them, and execute inline inside the active Browserless session
- [x] Inject caller-supplied parameters into the shared execution context with automatic restoration afterward
- [x] Support output mapping + `storeResult` so callee variables can hydrate parent workflow variables
- [x] Refresh React Flow node: load workflows from `useWorkflowStore`, allow manual UUID entry, JSON editors, disabled async toggle with helper text
- [x] Add tests (`runtime/instructions_test.go`, `client_test.go:TestExecuteWorkflowWorkflowCallInline`) and documentation (`docs/nodes/workflow-call.md`) with requirement `[REQ:BAS-NODE-WORKFLOW-CALL]`

**Acceptance Criteria:**
- Workflow call nodes compile only when workflow IDs are present and wait-for-completion is true
- Referenced workflows run inline without creating a new browser session, honoring branching/loop semantics
- Parameter overrides are scoped to the call and restored afterward
- Output mappings/`storeResult` surface callee data to downstream steps
- UI prevents invalid configurations and surfaces actual workflows
- Telemetry/logging captures workflow call duration + metadata for replay

---

### Phase 4: Control Flow (5-7 days) 🔀
**Priority:** MEDIUM
**Goal:** Enable data-driven testing, reduce workflow duplication

#### 4.1 Implement Conditional Node
**Status:** 🟢 Completed — runtime + UI branching shipped (2025-11-19)
**Effort:** 8-10 hours (COMPLEX - requires execution engine branching)
**Impact:** Dynamic workflows, error handling, feature detection

**Architecture Decision:**
- **Current branching:** Edge-level conditions (success/failure/assert_pass/assert_fail)
- **New branching:** Node-level conditions that evaluate BEFORE executing next node
- **Evaluation:** JavaScript expression, element existence check, variable comparison

**Implementation Checklist:**

**Phase 4.1.1: Execution Engine Updates (3-4 hours)**
- [x] Update `compiler/compiler.go` / `runtime/instructions.go` to introduce `StepConditional` + new params.
- [x] Teach `api/browserless/client.go` to interpret `if_true`/`if_false` edges via `ConditionResult` metadata.
- [x] Implement CDP `ExecuteConditional` covering expression, selector, and workflow-variable branches.
- [x] Persist outcomes inside `runtime.ConditionResult` so telemetry + artifacts capture branch decisions.

**Phase 4.1.2: Conditional Node (3-4 hours)**
- [x] Add `StepConditional` to compiler
- [x] Implement `conditionalConfig` struct in `runtime/instructions.go` with validation + defaults.
- [x] Support `if_true` / `if_false` handles + styling in React Flow (handles + edge metadata).
- [x] Add requirement entry `[REQ:BAS-NODE-CONDITIONAL]` and node palette/README coverage.

**Phase 4.1.3: Testing & Documentation (2-3 hours)**
- [x] Create Go workflow tests that branch TRUE/FALSE via mocked Browserless responses.
- [x] Add unit tests for expression + variable parsing failures in `instructions_test.go`.
- [x] Exercise selector/variable comparisons via CDP executor smoke tests (manual + unit coverage proxies).
- [x] Document conditional usage in README, node plan, and requirements traceability.

**Acceptance Criteria:**
- Conditional branching works for all condition types
- Execution follows correct branch (true/false)
- Nested conditionals work (conditional inside conditional)
- Tests cover all condition types and edge cases
- Visual representation clear in workflow builder

---

#### 4.2 Implement Loop Node
**Status:** 🟢 Completed — execution engine + UI shipped (2025-11-19)
**Effort:** 10-12 hours (VERY COMPLEX - requires sub-graph execution)
**Impact:** Data-driven testing, bulk operations, pagination

**Architecture Decision:**
- **Loop body:** Sub-graph of nodes that execute repeatedly
- **Iteration variable:** Injected into context as `{{loop.index}}`, `{{loop.item}}`
- **Loop types:** For-each (array), while (condition), repeat (count)
- **Max iterations:** Safety limit (default 100, configurable)

**Implementation Checklist:**

**Phase 4.2.1: Sub-Graph Execution (4-5 hours)**
- [x] Update compiler to identify loop body nodes
- [x] Implement loop body as isolated execution plan
- [x] Add loop iteration context:
  ```go
  type LoopContext struct {
      Index    int         `json:"index"`    // Current iteration (0-based)
      Item     interface{} `json:"item"`     // Current array item (for-each)
      IsFirst  bool        `json:"isFirst"`  // First iteration flag
      IsLast   bool        `json:"isLast"`   // Last iteration flag
      Total    int         `json:"total"`    // Total iterations (if known)
  }
  ```
- [x] Thread loop context through execution

**Phase 4.2.2: Loop Node (4-5 hours)**
- [x] Add `StepLoop` to compiler
- [x] Implement `loopConfig` struct:
  ```go
  type loopConfig struct {
      LoopType     string      `json:"loopType"`     // "forEach", "while", "repeat"
      ArraySource  string      `json:"arraySource"`  // Variable name containing array
      Condition    string      `json:"condition"`    // For while loops
      Count        int         `json:"count"`        // For repeat loops
      MaxIterations int        `json:"maxIterations"` // Safety limit
      BreakOn      string      `json:"breakOn"`      // Break condition
      ContinueOn   string      `json:"continueOn"`   // Continue condition
  }
  ```
- [x] Implement loop execution logic:
  - **For-each**: Iterate over array variable
  - **While**: Execute while condition true
  - **Repeat**: Execute N times
- [x] Add break/continue support via special edge conditions (`compiler.loopHandle*`, `client.evaluateOutgoingEdges` shipped 2025-11-18)
- [x] Add timeout per iteration (default 45s) and total loop timeout (default 5m) with runtime/client enforcement + UI controls
  - Runtime clamps iteration windows between 250ms-10m and total windows between 1s-30m so rogue loops cannot hang executions; builder inputs mirror these bounds.

**Phase 4.2.3: Testing & Documentation (2-3 hours)**
- [x] Create test workflow: Loop over test data CSV
- [x] Test nested loops
- [x] Test break/continue
- [x] Test max iteration safety limit
- [x] Document loop patterns and best practices
- [x] Add requirement: `[REQ:BAS-NODE-LOOP]`

**Acceptance Criteria:**
- For-each loops iterate over arrays correctly
- While loops respect condition
- Repeat loops execute N times
- Loop variables accessible in body nodes
- Break/continue work correctly
- Max iteration limit prevents infinite loops
- Per-iteration and total loop timeouts abort long-running workflows with descriptive errors
- Tests cover all loop types and edge cases

---

### Phase 5: Mobile & Browser Context (3-5 days) 📱
**Priority:** LOW (unless mobile testing is priority)
**Goal:** Enable mobile testing, API mocking, advanced browser features

#### 5.1 Implement Gesture Node (Mobile)
**Status:** 🟢 Completed — CDP touch + UI builder shipped (2025-11-19)
**Effort:** 6-8 hours
**Impact:** Mobile automation, touch interfaces

**Implementation Checklist:**
- [x] Add `StepGesture` to compiler
- [x] Implement `gestureConfig` struct:
  ```go
  type gestureConfig struct {
      GestureType string `json:"gestureType"` // "swipe", "pinch", "tap", "doubleTap", "longPress"
      Direction   string `json:"direction"`   // "up", "down", "left", "right"
      StartX      int    `json:"startX"`
      StartY      int    `json:"startY"`
      EndX        int    `json:"endX"`
      EndY        int    `json:"endY"`
      Distance    int    `json:"distance"`    // For swipe
      Scale       float64 `json:"scale"`      // For pinch (0.5 = pinch in, 2.0 = pinch out)
      DurationMs  int    `json:"durationMs"`
      Selector    string `json:"selector"`    // Target element
  }
  ```
- [x] Implement CDP touch event simulation (`Input.dispatchTouchEvent`)
- [x] Add gesture presets (swipe left/right, pinch to zoom, etc.)
- [x] Test with mobile viewport (go + vitest suites)
- [x] Add requirement: `[REQ:BAS-NODE-GESTURE-MOBILE]`

---

#### 5.2 Implement Rotate Node (Mobile)
**Status:** 🟢 Completed — orientation swap + UI shipped (2025-11-19)
**Effort:** 2-3 hours
**Impact:** Orientation change testing

**Implementation Checklist:**
- [x] Add `StepRotate` to compiler
- [x] Implement `rotateConfig` struct:
  ```go
  type rotateConfig struct {
      Orientation string `json:"orientation"` // "portrait", "landscape"
      Angle       int    `json:"angle"`       // 0, 90, 180, 270
      WaitForMs   int    `json:"waitForMs"`
  }
  ```
- [x] Implement CDP `Emulation.setDeviceMetricsOverride()` with orientation + telemetry debug context
- [x] Swap width/height on rotation and persist viewport state for future nodes
- [x] Add Rotate node UI component + palette entry + workflow builder registration
- [x] Add requirement: `[REQ:BAS-NODE-ROTATE-MOBILE]`
- [x] Add unit tests (Go + Vitest) covering defaults and palette updates

**Acceptance Criteria:**
- Rotate instructions compile with validated orientation + angle combos ✅ (`instructions_test.go`)
- CDP path issues `Emulation.setDeviceMetricsOverride` and tracks viewport swaps ✅ (`cdp/actions.go`)
- Builder exposes orientation + angle pickers with wait timing ✅ (`RotateNode.tsx`)
- Requirements + plan updated ✅ (this document + `requirements-traceability.md`)
- Manual end-to-end mobile run ⚠️ pending dedicated scenario fixture

---

#### 5.3 Implement Frame/iFrame Switch Node
**Status:** 🟢 Completed — frame stack + UI shipped (2025-11-20)
**Effort:** 4-5 hours
**Impact:** Embedded content, ads, widgets

**Implementation Checklist:**
- [x] Add `StepFrameSwitch` to compiler
- [x] Implement `frameSwitchConfig` struct:
  ```go
  type frameSwitchConfig struct {
      SwitchBy   string `json:"switchBy"`   // "index", "name", "selector", "url", "parent", "main"
      Index      int    `json:"index"`
      Name       string `json:"name"`
      Selector   string `json:"selector"`   // iframe selector
      URLMatch   string `json:"urlMatch"`
      TimeoutMs  int    `json:"timeoutMs"`
  }
  ```
- [x] Update CDP session to track frame context (frame stack + scoped evaluation helpers)
- [x] Implement CDP frame navigation + selection helpers (`frame_actions.go`)
- [x] Add requirement: `[REQ:BAS-NODE-FRAME-SWITCH]`

**Acceptance Criteria:**
- Frame switch nodes compile with validated selector/index/name/url/parent/main modes ✅ (`instructions.go`, `instructions_test.go`)
- Active frame context scopes DOM queries + Evaluate calls across nodes ✅ (`frame_context.go`, `interactions_basic.go`, `select_actions.go`, `scroll_actions.go`)
- CDP executor exposes `ExecuteFrameSwitch` with frame stack + debug info ✅ (`frame_actions.go`, `adapter.go`)
- Node palette + builder provide UI for frame switching ✅ (`FrameSwitchNode.tsx`, `NodePalette.tsx`, `WorkflowBuilder.tsx`)
- Requirement traceability updated ✅ (`docs/requirements-traceability.md`)
- Tests executed ✅ (`go test ./scenarios/browser-automation-studio/api/browserless/...`, `pnpm test -- --coverage`)

---

#### 5.4 Implement Cookie/Storage Nodes
**Status:** 🟢 Completed — cookie + storage primitives, UI nodes, and docs shipped (2025-11-21)
**Effort:** 4-5 hours (actual ~6h incl. docs/tests)
**Impact:** Session management, cache testing, localStorage manipulation

**Implementation Checklist:**
- [x] Add `StepSetCookie`, `StepGetCookie`, `StepClearCookie` to compiler
- [x] Implement cookie management via CDP `Network.setCookie()`, `Network.getCookies()`, `Network.clearBrowserCookies()` (`cookie_actions.go`)
- [x] Add `StepSetStorage`, `StepGetStorage`, `StepClearStorage` for localStorage/sessionStorage
- [x] Implement storage helpers via frame-aware evaluation (JSON validation, serialization) (`storage_actions.go`)
- [x] Store retrieved values/JSON payloads in workflow variables + Node UI storeAs fields
- [x] Add requirement + doc: `[REQ:BAS-NODE-COOKIE-STORAGE]` (`docs/nodes/cookie-storage.md`, `requirements-traceability.md`)

**Notes:** Runtime instructions now normalize same-site + storage formats, new Go unit tests cover validation paths, and React Flow nodes expose palette entries + configuration editors (Set/Get/Clear Cookie & Storage). NodePalette test count now sits at 33 entries with downstream interception nodes.

---

#### 5.5 Implement Network Mock/Intercept Node
**Status:** 🟢 Completed — fetch interception + UI shipped (2025-11-11)
**Effort:** ~9 hours (runtime plumbing + UI + docs/tests)
**Impact:** API testing, offline mode, error simulation

**Implementation Checklist:**
- [x] Add `StepNetworkMock` + config struct to compiler/runtime with validation + tests
- [x] Implement CDP `Fetch.enable()` listener w/ rule registry, regex/glob support, and response/abort/delay handling (`network_mock.go`)
- [x] Persist rules as React Flow node (`NetworkMockNode`), palette card, builder wiring, and vitest coverage
- [x] Add requirement + doc: `[REQ:BAS-NODE-NETWORK-MOCK]` (`docs/nodes/network-mock.md`, `requirements-traceability.md`)
- [x] Document limitations (CORS, SPA caching) + performance guardrails (only enable Fetch when rules exist)

**Notes:**
- Runtime clamps status codes + delays, normalizes abort reasons, and stores body/header previews in execution artifacts.
- CDP session now syncs interception rules across targets/tabs automatically and logs when a rule matches.
- UI exposes pattern helper text (`regex:` prefix), method picker, abort reasons, headers text area, and body editor to keep workflows declarative.

---

### Phase 6: UI Reorganization (2-3 days) 🎨
**Priority:** HIGH (once node count > 15)
**Goal:** Improve discoverability, reduce cognitive load

#### 6.1 Implement Collapsible Categories
**Status:** 🟢 Completed — collapsible palette shipped with persistence + accessibility polish (2025-11-22)
**Effort:** 6-8 hours
**Impact:** Essential for UX when 20+ nodes exist

**Implementation Checklist:**

**Phase 6.1.1: Category Data Structure (1-2 hours)**
- [x] Create `ui/src/constants/nodeCategories.ts` with typed `WorkflowNodeDefinition` + `NodeCategory` collections, lucide icons, keywords, and grouped node IDs:
  ```typescript
  export interface NodeCategory {
      id: string;
      label: string;
      icon: React.ComponentType;
      description: string;
      defaultExpanded: boolean;
      nodes: string[]; // Node type IDs
  }

  export const NODE_CATEGORIES: NodeCategory[] = [
      {
          id: 'navigation',
          label: 'Navigation',
          icon: Compass,
          description: 'Navigate pages and scroll',
          defaultExpanded: false,
          nodes: ['navigate', 'scroll']
      },
      {
          id: 'interaction',
          label: 'Interaction',
          icon: MousePointer,
          description: 'Click, type, and interact with elements',
          defaultExpanded: true,
          nodes: ['click', 'type', 'select', 'hover', 'focus', 'dragDrop', 'uploadFile']
      },
      // ... other categories
  ];
  ```

**Phase 6.1.2: Category UI Component (2-3 hours)**
- [x] Update `ui/src/components/NodePalette.tsx` to hydrate default-expanded categories, store the selection in localStorage, and expose `toggleCategory` with ARIA wiring:
  ```tsx
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
      new Set(NODE_CATEGORIES.filter(c => c.defaultExpanded).map(c => c.id))
  );

  const toggleCategory = (categoryId: string) => {
      setExpandedCategories(prev => {
          const next = new Set(prev);
          if (next.has(categoryId)) {
              next.delete(categoryId);
          } else {
              next.add(categoryId);
          }
          return next;
      });
  };
  ```
- [x] Implement category headers with expand/collapse controls, icons, and `aria-expanded`
- [x] Add smooth max-height transitions for bodies while keeping DOM nodes mounted for drag support
- [x] Persist expanded state to `bas.palette.categories` in localStorage so user preference survives reloads

**Phase 6.1.3: Testing & Polish (1-2 hours)**
- [x] Verify expand/collapse performance + smoothness with full 33-node palette
- [x] Ensure category headers are focusable buttons with `Enter`/`Space` toggle and aria attributes
- [x] Confirm expanded state persists between reloads and that search locks categories open for clarity
- [x] Cover via `pnpm test -- --coverage`

**Acceptance Criteria:**
- Categories collapse/expand smoothly ✅ max-height transitions + persisted default state
- Default expansion state matches user expectations ✅ Navigation + Pointer default to expanded via config
- State persists across page refreshes ✅ stored in `bas.palette.categories`
- Keyboard navigation works ✅ buttons with aria metadata + disabled state when search is active
- Performance is good (no lag on expand) ✅ verified with 33 nodes + vitest smoke coverage

---

#### 6.2 Implement Search/Filter
**Status:** 🟢 Completed — fuzzy search + highlighting shipped (2025-11-22)
**Effort:** 3-4 hours
**Impact:** Power users find nodes instantly

**Implementation Checklist:**
- [x] Add search input above categories with CMD/CTRL+K shortcut + ESC to clear
- [x] Implement fuzzy search (label, description, keywords) with subsequence fallback + highlight function
- [x] Auto-expand matching categories while search is active and disable toggle buttons for clarity
- [x] Highlight matched substrings using `<mark>` and show "No nodes match" empty state
- [x] Clear search on node drag (so quick drags are frictionless) and rehydrate from persisted state

**Acceptance Criteria:**
- Search finds nodes by name or description ✅ fuzzy helper covers label/description/keywords
- Fuzzy matching works (typos, partial matches) ✅ subsequence matcher + tests around "cookie" results
- Results update in real-time ✅ controlled input + derived category filtering
- Keyboard shortcut works ✅ global `keydown` handler focuses the search field on Cmd/Ctrl+K

---

#### 6.3 Implement Favorites/Recents
**Status:** 🟢 Completed — Quick Access (favorites + recents) shipped (2025-11-22)
**Effort:** 4-5 hours
**Impact:** Power users have <3 click access to common nodes

**Implementation Checklist:**
- [x] Add collapsible Quick Access section ahead of categories with History icon + status text
- [x] Track last 5 unique recents and enforce a 5-item favorite limit (oldest dropped) with clear button
- [x] Add star toggles on cards (`aria-pressed`) and persist favorites/recents via localStorage keys
- [x] Render favorites first, hide duplicates between favorites/recents, and reuse NodeCard for drag consistency
- [x] Document behavior in plan + cover via vitest quick-access regression

**Acceptance Criteria:**
- Recently used nodes appear in Quick Access ✅ recents list populates immediately after drag (tested)
- Favorited nodes persist across sessions ✅ stored in `bas.palette.favorites`
- Quick Access section is collapsible ✅ toggle button with chevron + aria-expanded
- Limit enforced (5 favorites, 5 recents) ✅ runtime clamp + vitest expectation

---

## 📊 Progress Tracking

### Phase Completion Status

| Phase | Status | Progress | Est. Completion |
|-------|--------|----------|-----------------|
| Phase 1: Unblock Testing | 🟢 Completed | 3/3 tasks (playbook rerun pending) | 2025-11-16 |
| Phase 2: Core UX Nodes | 🟢 Completed | 4/4 nodes | 2025-11-18 |
| Phase 3: Advanced Interaction | 🟢 Completed | 5/5 tasks | 2025-11-18 |
| Phase 4: Control Flow | 🟢 Completed | 2/2 tasks | 2025-11-19 |
| Phase 5: Mobile & Context | 🟢 Completed | 5/5 tasks | 2025-11-21 |
| Phase 6: UI Reorganization | 🟢 Completed | 3/3 tasks | 2025-11-22 |

### Node Implementation Status

#### ✅ Implemented (33 nodes)
- **Navigation & Context:** Navigate, Scroll, Tab Switch, Frame Switch, Rotate, Gesture
- **Pointer & Forms:** Click, Hover, Drag & Drop, Focus, Blur, Type, Select, Upload File, Shortcut, Keyboard
- **Data & Logic:** Extract, Set Variable, Use Variable, Script/Evaluate, Conditional, Loop, Workflow Call
- **Validation & Observability:** Wait, Screenshot, Assert
- **Storage & Network:** Set Cookie, Get Cookie, Clear Cookie, Set Storage, Get Storage, Clear Storage, Network Mock

#### 🔴 Phase 1: Critical (2 nodes)
- [x] Script/Evaluate
- [x] Keyboard

#### 🟡 Phase 2: Core UX (4 nodes + variables)
- [x] Hover
- [x] Scroll
- [x] Select
- [x] Set Variable
- [x] Use Variable (interpolation system)

#### 🟠 Phase 3: Advanced (5 nodes)
- [x] Focus
- [x] Blur
- [x] Drag & Drop
- [x] Tab/Window Switch
- [x] Upload File

#### 🟣 Phase 4: Control Flow (2 nodes)
- [x] Conditional
- [x] Loop (now includes per-iteration + total timeout enforcement)

#### 🟤 Workflow Reuse (new)
- [x] Workflow Call (inline execution, parameter passing, output mapping)

#### 🔵 Phase 5: Mobile & Context (7 nodes)
- [x] Gesture
- [x] Rotate
- [x] Frame Switch
- [x] Set Cookie
- [x] Get Cookie
- [x] Set Storage
- [x] Network Mock

#### 🟢 Phase 6: UI (3 features)
- [x] Collapsible Categories
- [x] Search/Filter
- [x] Favorites/Recents (Quick Access)

**Total:** 33 implemented (exceeds 27-node target) + 3 palette improvements. Remaining roadmap items are workflow-call execution + follow-up tests.

### Documentation Status (2025-11-13)

- Added `docs/NODE_INDEX.md` as the canonical cross-reference between palette categories, compiler step types, and documentation links.
- Authored dedicated READMEs for the control-flow nodes delivered in Phase 4: `docs/nodes/conditional.md` and `docs/nodes/loop.md` (in addition to the earlier workflow-call, network-mock, and cookie/storage guides).
- Documented the highest-traffic primitives (`docs/nodes/navigate.md`, `docs/nodes/click.md`) so onboarding engineers have end-to-end guidance for the entry/interaction steps.
- Extended coverage to other staple interactions: `docs/nodes/scroll.md`, `docs/nodes/hover.md`, `docs/nodes/type.md`, and `docs/nodes/select.md` now outline configuration, runtime behavior, and examples for their respective requirements.
- Rounded out the Assertions & Form helpers with `docs/nodes/wait.md`, `docs/nodes/assert.md`, `docs/nodes/focus.md`, and `docs/nodes/blur.md`, leaving the remaining palette entries as the only outstanding docs.
- Captured advanced interactions (`docs/nodes/drag-drop.md`, `docs/nodes/shortcut.md`, `docs/nodes/keyboard.md`, `docs/nodes/screenshot.md`) so every high-traffic pointer/keyboard/artifact node now has a dedicated reference.
- Remaining nodes still need per-node READMEs; the Node Index Reference column shows the current gaps so tech writers can divide the work.

---

### Latest Validation — 2025-11-13

- **Node inventory:** `rg -c "type: '" ui/src/constants/nodeCategories.ts` confirms **33** node definitions with matching compiler/runtime support. No drift detected between palette + backend wiring.
- **Go unit coverage:** `go test ./... -cover` now runs cleanly (35.2% overall; **runtime package 56.4%** after the new `ExecutionContext`/interpolation tests). This keeps Phase 2.4 variable-system work green while highlighting the remaining coverage gap.
- **UI tests:** `pnpm test -- --coverage` now shells through `ui/scripts/run-vitest.sh`, which runs eight Vitest projects sequentially (stores, components-core, components-palette, workflow-builder, etc.). Each project constrains its own pool/threads so the full suite completes without revving the heap beyond ~2.6 GB. The aggregated requirement report (`coverage/vitest-requirements.json`) reflects 136 Vitest assertions with 0 failures.
- **Open items:** Browserless integration playbooks (`test/playbooks/replay/render-check.json`, `test/playbooks/ui/projects/*`) and perf/UX telemetry remain pending on infrastructure fixes.

---

## 🧪 Testing Strategy

### Test Coverage Requirements

Each node must have:

1. **Go Unit Tests** (`api/browserless/runtime/instructions_test.go`)
   - Valid config → successful instruction
   - Invalid config → error
   - Missing required params → error
   - Edge cases (empty strings, negative numbers, etc.)
   - Requirement tag: `[REQ:BAS-NODE-{TYPE}-{VALIDATION}]`

2. **Go Integration Tests** (`api/browserless/client_test.go` or CDP tests)
   - Mock Browserless/CDP responses
   - Execute instruction → verify outcome
   - Timeout handling
   - Error propagation

3. **Vitest Tests** (`ui/src/components/__tests__/NodePalette.test.tsx`)
   - Node appears in palette
   - Draggable
   - Category assignment correct
   - Search finds node

4. **BAS Workflow Test** (`test/playbooks/nodes/{node-type}.json`)
   - Real browser execution
   - Verify telemetry captured
   - Verify screenshots taken
   - Verify execution success

### Requirement Tagging Convention

```go
// Go tests
func TestHoverNode_TriggersDropdown(t *testing.T) {
    t.Run("[REQ:BAS-NODE-HOVER-INTERACTION] triggers CSS :hover state", func(t *testing.T) {
        // test code
    })
}

// Vitest tests
describe('HoverNode [REQ:BAS-NODE-HOVER-INTERACTION]', () => {
    it('appears in Interaction category', () => {
        // test code
    });
});
```

### Integration Test Workflow

### Running the UI suite locally

- `pnpm test -- --coverage` wraps `ui/scripts/run-vitest.sh`, which sequentially executes the Vitest projects (`stores`, `components-core`, `components-palette`, `workflow-builder`).
- Each project has its own pool/thread configuration in `vite.config.ts::test.projects`, so high-memory suites (WorkflowBuilder) run in an isolated fork with `max-old-space-size=8192` while lighter suites keep multithreaded pools.
- The helper script captures the per-project requirement reports (`coverage/vitest-requirements-*.json`), merges them after the loop, and emits a single `coverage/vitest-requirements.json` for the scenario-wide requirement tracking.

After each phase:
```bash
# Run unit tests
cd api && go test -v ./...

# Run UI tests
cd ui && pnpm test

# Run integration tests
./test/run-tests.sh unit integration

# Verify requirement coverage
node scripts/requirements/report.js --scenario browser-automation-studio --format markdown
```

---

## 🚧 Known Challenges & Mitigations

### Challenge 1: Variable System Complexity
**Risk:** Threading execution context through pipeline is complex
**Mitigation:**
- Start with simple in-memory map
- Persist context in artifacts for debugging
- Defer global/project variables to future phase

### Challenge 2: Loop Node Sub-Graph Execution
**Risk:** Nested loops, break/continue logic is error-prone
**Mitigation:**
- Implement iterative (not recursive) loop execution
- Add max iteration safety limit (default 100)
- Extensive testing with nested loops

### Challenge 3: Conditional Node Branching
**Risk:** Complex branching logic may conflict with existing edge conditions
**Mitigation:**
- Use distinct edge types (`if_true`, `if_false` vs `success`, `failure`)
- Update compiler validation to prevent ambiguous branching
- Document precedence rules clearly

### Challenge 4: Drag & Drop Browser Compatibility
**Risk:** HTML5 drag-drop API simulation is unreliable
**Mitigation:**
- Implement CDP mouse event fallback
- Test with multiple drag-drop implementations (HTML5, React DnD, etc.)
- Document limitations and workarounds

### Challenge 5: Network Mock Performance
**Risk:** Intercepting all requests may slow execution
**Mitigation:**
- Only enable interception when mock nodes present
- Use URL pattern filtering at CDP level
- Document performance considerations

---

## 📚 Documentation Requirements

### For Each New Node

1. **Node README** (`docs/nodes/{node-type}.md`)
   - Purpose and use cases
   - Configuration parameters (with types and defaults)
   - Examples (simple + advanced)
   - Limitations and gotchas
   - Related nodes

2. **API Documentation** (in Go file comments)
   - Struct field documentation
   - CDP method calls used
   - Error conditions
   - Performance notes

3. **Requirement Entry** (`requirements/{category}/{node-type}.json`)
   - Requirement ID
   - Validation references (test files)
   - Criticality
   - Status

4. **Update Node Index** (`docs/NODE_INDEX.md`)
   - Add node to category listing
   - Link to node README
   - Show example usage

---

## 🎯 Success Criteria (Final)

### Functional
- [x] 33 nodes implemented and tested (compiler `supportedStepTypes` + NodePalette definitions + unit/vitest coverage)
- [ ] 100% integration tests passing *(pending — `./test/run-tests.sh integration` still blocked on Browserless fixtures)*
- [ ] All workflows in `test/playbooks/` execute successfully *(needs full playbook sweep once scenarios are runnable)*
- [ ] Requirements coverage: 100% for implemented features *(latest unit suite exercises 13 requirements; more remain)*

### Quality
- [ ] Go test coverage: >80% for new code *(current aggregate ~35.2% from `go test ./...`)*
- [ ] Vitest test coverage: >75% for new UI components *(coverage instrumentation not yet enabled in Vitest run)*
- [ ] No P0 bugs in production workflows *(requires production telemetry review)*
- [x] All nodes documented with examples *(dedicated READMEs now exist for all 33 nodes - see `docs/NODE_INDEX.md` for links)*

### UX
- [ ] Time to find node: <5 seconds for new users *(needs moderated usability test)*
- [x] Node palette organized into 7 categories (see React tests in `NodePalette.test.tsx`)
- [x] Search finds nodes by name or description (`NodePalette` fuzzy search test covers label/description/keyword matching)
- [x] Favorites/recents improve power user workflow (Quick Access tests verify recents + favorites persistence/limits)

### Performance
- [ ] Workflow execution time: <1s overhead per node *(pending telemetry after integration run)*
- [ ] UI responsive: <100ms category expand/collapse *(requires browser profiling/instrumentation)*
- [ ] Search results: <50ms for 27 nodes *(needs runtime metric capture)*
- [ ] No memory leaks in long-running workflows *(awaiting extended soak test)*

---

## 🔄 Maintenance & Updates

### When Adding New Nodes in Future

1. **Classification:**
   - Determine category (or create new if needed)
   - Assign priority tier (1-4)

2. **Implementation:**
   - Follow phase structure (compiler → runtime → CDP → UI)
   - Copy implementation checklist from similar node
   - Add all required tests

3. **Documentation:**
   - Create node README
   - Update NODE_INDEX.md
   - Add requirement entry
   - Update this plan

4. **Review:**
   - Code review focuses on consistency with existing nodes
   - UX review focuses on discoverability and clarity
   - Test review focuses on coverage and edge cases

### Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-11-10 | Initial plan | BAS Team |

---

## 📞 Getting Help

### Questions or Blockers?

- **Architecture decisions:** Discuss in team chat before implementing
- **CDP/Browserless issues:** Check CDP DevTools Protocol docs
- **Test failures:** Review `test/artifacts/` logs for debugging
- **UX questions:** Review Figma mockups or create quick prototype

### Useful Resources

- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)
- [Playwright Node Types](https://playwright.dev/docs/api/class-page) (inspiration)
- [Puppeteer Actions](https://pptr.dev/) (reference implementation)
- [BAS Testing Guide](./testing/architecture/PHASED_TESTING.md)
- [BAS Requirements](./requirements-traceability.md)

---

**Remember:** This is a living document. Update it as you complete tasks, encounter challenges, or discover better approaches. Keep it synchronized with reality.

**Next Review:** After Phase 1 completion
