# Experience Architecture Audit

Last updated: 2026-03-11 (Iteration 2)
Phase: 18 (React Coherence + Experience Architecture + Navigation Integrity)

## Scenario Purpose Statement

**Purpose**: This scenario helps prompt-manager's meta optimization team and AI agents **validate steer skill interoperability and development tooling correctness** so they can **detect cross-skill conflicts before expensive agent iteration loops** and **ensure development tools (auditor, test-genie, completeness-scoring) produce correct results against known-good references**.

---

## Core Personas & Primary Jobs

### Persona 1: Meta Optimization Agent (Primary)

The primary consumer - uses CLI to detect skill issues and tooling regressions. Runs automated validation loops.

| Job | Description | Priority |
|-----|-------------|----------|
| Validate Reference | Run all structural and CLI assertions against a reference scenario | High |
| Detect Drift | Check if connected skills have changed since configuration | High |
| Review Validation Report | See pass/fail status for all skill connections on a reference | Medium |

### Persona 2: Skill Author

An AI agent or human improving steer skills via prompt-manager. Needs to verify their skill works correctly.

| Job | Description | Priority |
|-----|-------------|----------|
| Configure Expectations | Define structural/CLI expectations for their skill on a reference | High |
| Check Coverage | See which files/folders their skill covers vs gaps | Medium |
| Identify Conflicts | See if their skill conflicts with other connected skills | Medium |

### Persona 3: Operations/Monitoring User

Human developer checking overall system health.

| Job | Description | Priority |
|-----|-------------|----------|
| View Reference Health | Quick status of all references and their validation state | High |
| Identify Failing Skills | See which skill connections are broken | Medium |
| Debug Validation Failures | Understand why a specific assertion failed | Medium |

---

## Current vs Ideal Flows

### Flow 1: View Reference Health (Operations User)

**Current Flow:**
1. Visit UI → Dashboard shows reference cards with name/slug/template/path
2. Cards show no validation status (pass/fail/unknown)
3. No way to see skill connections or validation results from UI
4. Must use CLI for any actual validation

**Ideal Flow:**
1. Visit UI → Dashboard shows references with health indicators (✓/✗/⚠)
2. Click reference card → See detail view with connected skills and validation status
3. See recent validation history and any failures
4. One-click to run validation from UI (or link to CLI command)

**Friction Points:**
- **Discoverability (Critical)**: UI shows references but no validation/skill connection data - the core purpose of the scenario is invisible from the UI
- **Cognitive**: User must switch to CLI to do anything meaningful
- **No detail view**: Reference cards don't expand or navigate anywhere

### Flow 2: Configure Skill Expectations (Skill Author)

**Current Flow:**
1. CLI only: `dtv connection connect --reference-id X --skill-id Y`
2. CLI: `dtv expectation create-structural --connection-id Z ...`
3. CLI: `dtv expectation create-cli --connection-id Z ...`
4. No visual feedback on what's configured
5. No validation preview before committing

**Ideal Flow:**
1. UI: See reference → Add Skill Connection → Configure expectations visually
2. See existing expectations in context (which files/folders)
3. Test individual expectations before saving
4. See coverage map showing where the skill touches the reference

**Friction Points:**
- **Mechanical (High)**: Many CLI commands required for a single connection setup
- **No UI support**: All configuration is CLI-only despite having a React UI
- **No preview**: Can't test expectations without creating them permanently

### Flow 3: Run Validation (Meta Optimization Agent)

**Current Flow:**
1. CLI: `dtv validate --reference-id X` (not yet implemented)
2. Results returned to CLI as JSON
3. No persistent history of validation runs

**Ideal Flow:**
1. CLI: `dtv validate <reference>` with human-readable output
2. Results persisted to database with timestamp
3. Can view validation history in UI
4. Drift detection integrated into validation flow

**Friction Points:**
- **Not yet implemented**: Validation endpoints/CLI commands don't exist yet
- **No history**: Validation results aren't persisted for trend analysis

---

## Navigation Architecture

### Current State (After Phase 18 Iteration 2)

The UI now has two-level routing with React Router:

```
[Dashboard] (/dashboard)          ← Overview with reference list
    └── [Reference Detail] (/references/:slug) ← Full detail with connections
```

### Navigation Controls

1. **Reference cards are clickable**: Navigate to `/references/:slug` detail page
2. **Back button**: In header on detail page, returns to dashboard
3. **Footer link**: Alternative back navigation in footer
4. **Browser history**: Standard back/forward navigation works correctly

### Navigation Issues Resolved

1. ✅ **Reference cards now navigate**: Click any card to see full detail view
2. ✅ **Detail page with connections**: Shows full reference info + connected skills
3. ✅ **Back navigation**: Clear path back to dashboard from detail page
4. ✅ **Deep linking supported**: Direct URL access to `/references/:slug` works

### Proposed Future Navigation

```
[Dashboard]                      ← Overview with health summaries ✅
    └── [Reference Detail]       ← Connected skills, validation status ✅
            └── [Skill Connection] ← Expectations config, assertion results (future)
                    └── [Validation Result] ← Detailed pass/fail with context (future)
```

Remaining navigation goals:
- Add skill connection detail views (requires expectation API handlers)
- Add validation result views (requires validation API handlers)

---

## Friction Analysis Summary

| Category | Issue | Severity | Location |
|----------|-------|----------|----------|
| Discoverability | Core functionality (validation/connections) not visible in UI | Critical | Dashboard |
| Mechanical | Must use CLI for all operations except viewing reference list | High | Entire UI |
| Cognitive | UI doesn't communicate scenario's purpose effectively | Medium | Dashboard |
| Navigation | Reference cards are display-only, no drill-down | Medium | Dashboard |
| Missing Feature | No validation endpoints/handlers exposed via API | Blocking | API |

---

## Improvements for Phase 18

### Implemented - Iteration 1

1. **Reference Card Expansion** - Made reference cards expandable with inline skill connection display
2. **Skill Connection Status Display** - Show which skills are connected to each reference with basic status
3. **API Client Extensions** - Added skill connection fetching to UI
4. **Skill Handler Wiring** - Fixed connections endpoint returning 404 (handler was never registered)

### Implemented - Iteration 2

1. **React Router Navigation** - Added `react-router-dom` with BrowserRouter for proper navigation
2. **Reference Detail Page** - Created dedicated page at `/references/:slug` showing:
   - Full reference info (name, slug, template, path, description, ID, timestamps)
   - Connected skills with version and content hash
   - CLI quick reference commands
3. **Navigation Affordances**:
   - Reference cards are now clickable links to detail pages
   - Back button in header returns to dashboard
   - Footer link provides secondary back navigation
   - Browser history navigation works correctly
4. **Component Organization**:
   - Created `/pages/Dashboard.tsx` extracted from App.tsx
   - Created `/pages/ReferenceDetail.tsx` for detail view
   - App.tsx now purely handles routing
5. **Button Ghost Variant** - Added ghost variant to button component for back navigation
6. **Updated Selectors** - Added navigation-related selectors for automation

### Implemented - Iteration 3

1. **Component Extraction** - Extracted reusable UI primitives:
   - `components/ui/card.tsx` - Card, CardHeader, CardContent, CardFooter with variants
   - `components/ui/badge.tsx` - Badge primitive with CVA variants (default, primary, success, warning, danger)
2. **Keyboard Shortcuts** - Centralized shortcut manager with iframe-bridge integration:
   - `hooks/useKeyboardShortcut.ts` - Core hook with emitShortcutIntent for iframe interop
   - `hooks/useNavigationShortcuts.ts` - Pre-configured navigation shortcuts
   - Dashboard: Press `r` to refresh data
   - Detail page: Press `r` to refresh, `Escape` or `h` to go back to dashboard
3. **Badge Component Integration** - Dashboard and Detail pages now use Badge component for template and connection count badges

### Deferred to Future Loops

1. **Full validation UI** - Running validations from UI (requires API handlers first)
2. **Expectation configuration UI** - Visual config of structural/CLI expectations
3. **Coverage map visualization** - File tree showing skill coverage
4. **Validation history** - Persistent results with trend view

---

## Technical Notes

### Current UI Architecture (After Iteration 3)

- **State**: React Query for server state, local useState for UI state
- **Components**: Organized into `/pages/`, `/components/`, `/hooks/`
- **Styling**: Tailwind with semantic tokens, CVA for primitives (Button, Badge)
- **Routing**: React Router with BrowserRouter, 2 routes (/, /references/:slug)
- **API Integration**: Health + References CRUD + Connections list (5/6 endpoints)
- **Keyboard Shortcuts**: Centralized via useKeyboardShortcut with iframe-bridge integration

### File Structure

```
src/
├── App.tsx              # Router configuration
├── main.tsx             # Entry point with providers
├── pages/
│   ├── Dashboard.tsx    # Reference list view
│   └── ReferenceDetail.tsx # Reference detail view with connections
├── components/
│   ├── ErrorBoundary.tsx
│   ├── Layout.tsx       # Shared header/main/footer layout
│   └── ui/
│       ├── badge.tsx    # Badge primitive with CVA variants
│       ├── button.tsx   # Button primitive with CVA variants
│       ├── card.tsx     # Card component family
│       ├── CopyableCode.tsx # CLI command display with copy
│       └── HealthIndicator.tsx # API health status display
├── hooks/
│   ├── index.ts         # Hook re-exports
│   └── useKeyboardShortcut.ts # Centralized keyboard shortcuts
├── lib/
│   ├── api.ts           # API client functions
│   └── utils.ts         # Utilities (cn, formatDate)
└── consts/
    └── selectors.ts     # Automation selectors
```

### UI Gaps vs PRD (Updated)

| PRD Target | UI Support | Status |
|------------|------------|--------|
| OT-P0-001 Reference Registry | List + Detail view | **In Progress** |
| OT-P0-002 Skill Connections | View connections per reference | **In Progress** |
| OT-P0-003 Drift Detection | None | Not started |
| OT-P0-004 Structural Expectations | None | Not started |
| OT-P0-005 CLI Expectations | None | Not started |
| OT-P0-006-008 Validation | None | Not started |
| OT-P1-007 Dashboard UI | Reference list with connection counts | **In Progress** |
| OT-P1-008 Skill Detail View | None | Not started |

The UI now supports viewing references and their connections. All other functionality (validation, expectations) still requires CLI.
