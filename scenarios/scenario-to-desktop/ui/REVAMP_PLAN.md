# Generate Desktop App Page Revamp Plan

## Current State Analysis

### Issues Identified

1. **DebugJsonModal Crash (React Error #185)**
   - Error: "Cannot update component while rendering a different component"
   - Root cause: The modal subscribes to the full pipeline store state during render. With polling every 2 seconds, store updates can trigger re-renders while the component is still rendering, causing this error.
   - Location: `components/layout/DebugJsonModal.tsx:36-55`

2. **Double Padding Issue**
   - Section cards add padding (`px-6 pb-6`) in child content
   - Content within sections also adds padding
   - Results in excessive whitespace that looks unprofessional
   - Example: `GeneratorPage.tsx:177-183` - `<div className="px-6 pb-6">` wraps already-padded content

3. **Sections Are Placeholders**
   - Most sections just display "managed in Configuration section above"
   - All actual functionality is crammed into `ConfigurationSection` → `GeneratorForm` (1078 lines)
   - Sections: Bundle, Preflight, Generate, Build, Smoke Test, Distribution are essentially empty shells

4. **GeneratorForm is Monolithic**
   - 1078 lines in a single component
   - 30+ useState hooks
   - Multiple concerns mixed: form state, API calls, validation, preflight, bundle, UI rendering
   - Difficult to maintain and extend

5. **Pipeline ID Shows "-"**
   - This is expected behavior when no pipeline has been started
   - However, the UI doesn't make it clear that a pipeline needs to be started first

6. **Unused/Deprecated Code**
   - `StatsPanel.tsx` - Not imported anywhere (0 imports found)
   - `TemplateGrid.tsx` - Only used by `TemplateModal.tsx` and its test
   - `pipelineStore.error` - Marked @deprecated but still present

### What's Working Well

- **Two-column layout with sticky sidebar** - Already implemented correctly
- **Collapsible sidebar** - Implemented with localStorage persistence
- **Pipeline status tracking** - Store has comprehensive selectors for stage status
- **Shared section components** - `SectionCard` and `SectionHeader` exist and work well
- **Server-side state persistence** - `useScenarioState` hook is well-designed

---

## Revamp Plan

### Phase 1: Fix Critical Bugs

#### 1.1 Fix DebugJsonModal Crash
**File:** `components/layout/DebugJsonModal.tsx`

**Current Problem:**
```tsx
function DebugJsonModalContent({ ... }) {
  // Store subscription happens during render - causes error #185
  const storeState = usePipelineStore((state) => ({
    scenarioName: state.scenarioName,
    // ... 15 more fields selected during render
  }));
```

**Solution:** Use `useRef` to snapshot the state once on mount, preventing store updates from triggering re-renders during the modal's render cycle:

```tsx
function DebugJsonModalContent({ onClose, copied, setCopied }: DebugJsonModalContentProps) {
  // Snapshot state on mount to avoid render-during-render issues
  const [snapshotState] = useState(() => usePipelineStore.getState());

  // Stable selector that only updates on explicit refresh
  const storeState = useMemo(() => ({
    scenarioName: snapshotState.scenarioName,
    pipelineId: snapshotState.pipelineId,
    // ... rest of fields
  }), [snapshotState]);

  // Optional: Add refresh button to get latest state
  const handleRefresh = () => {
    setSnapshotState(usePipelineStore.getState());
  };
```

Alternative: Wrap the JSON stringify in `useMemo` with a stable dependency:
```tsx
const storeState = usePipelineStore.getState(); // Get once, no subscription
const jsonString = useMemo(() =>
  JSON.stringify(storeState, safeReplacer, 2),
  [storeState]
);
```

### Phase 2: Fix Padding Issues

#### 2.1 Standardize Section Content Padding
**Files:** `components/sections/shared/SectionCard.tsx`, all section components

**Strategy:**
- `SectionCard` should NOT add any internal padding
- Each section's content is responsible for its own padding
- Use consistent padding class: `p-6` for standard content, `px-6 pb-6` when header is separate

**Changes:**
1. Update `SectionCard` to remove default padding assumptions
2. Update section placeholder content in `GeneratorPage.tsx` to remove redundant padding wrappers
3. Ensure `GeneratorForm` content doesn't double-pad

### Phase 3: Populate Section Content

#### 3.1 Move Functionality to Appropriate Sections

**Current:** Everything in `ConfigurationSection` → `GeneratorForm`

**Target Architecture:**

| Section | Content | Source |
|---------|---------|--------|
| Configuration | Scenario selection, app metadata, framework/template | Extract from GeneratorForm |
| Bundle | Bundle manifest, runtime selection | Extract `BundledRuntimeSection` logic |
| Preflight | Environment validation, secrets | Extract `BundledPreflightSection` logic |
| Generate | Trigger generation, view output path | New component |
| Build | Platform selection, signing config, trigger build | Extract from GeneratorForm |
| Smoke Test | Run smoke tests, view results | New component using pipeline store |
| Distribution | Upload artifacts | Already has `DistributionUploadSection` |

#### 3.2 Create New Section Components

**New files to create:**
```
components/sections/
├── configuration/
│   └── ConfigurationSection.tsx (refactored - metadata only)
├── bundle/
│   └── BundleSection.tsx (new)
├── preflight/
│   └── PreflightSection.tsx (new)
├── generate/
│   └── GenerateSection.tsx (new)
├── build/
│   └── BuildSection.tsx (new)
├── smoketest/
│   └── SmokeTestSection.tsx (new)
├── distribution/
│   └── DistributionSection.tsx (new)
├── shared/
│   ├── SectionCard.tsx (existing)
│   └── SectionHeader.tsx (existing)
└── index.ts (export all)
```

#### 3.3 Create Form Context

To avoid prop drilling and enable section components to share form state:

**New file:** `components/generator/GeneratorFormContext.tsx`

```tsx
interface GeneratorFormContextValue {
  // Scenario state
  scenarioName: string;
  setScenarioName: (name: string) => void;

  // Deployment config
  deploymentMode: DeploymentMode;
  setDeploymentMode: (mode: DeploymentMode) => void;

  // Platform config
  platforms: PlatformSelection;
  setPlatforms: (platforms: PlatformSelection) => void;

  // ... other shared state

  // Actions
  runPreflight: () => Promise<void>;
  generateApp: () => Promise<void>;
  buildInstallers: () => Promise<void>;
}

export const GeneratorFormContext = createContext<GeneratorFormContextValue | null>(null);
export const useGeneratorForm = () => useContext(GeneratorFormContext);
```

### Phase 4: Enhance Sidebar UX

#### 4.1 Improve Pipeline Status Display

**File:** `components/layout/SidebarHeader.tsx`

**Changes:**
- When no pipeline exists, show "No active pipeline" instead of just "-"
- Add visual indicators for each pipeline state
- Show elapsed time for running pipelines
- Show completion time for finished pipelines

#### 4.2 Improve Section Status Indicators

**File:** `components/layout/SidebarNavigation.tsx`

**Changes:**
- Add progress indicators for running stages
- Show stage-specific status messages (e.g., "Bundling: 3/5 packages")
- Highlight blocked sections (those waiting for prerequisites)

### Phase 5: Remove Unused Code

#### 5.1 Files to Delete
- `components/StatsPanel.tsx` - 0 imports found
- `components/__tests__/TemplateGrid.test.tsx` - Test for potentially unused component

#### 5.2 Files to Evaluate for Removal
- `components/TemplateGrid.tsx` - Only used in `TemplateModal`, may be replaceable with simpler component
- `pipelineStore.error` field - Marked @deprecated, should migrate all usages to `errorInfo`

### Phase 6: Screaming Architecture Refactor

#### 6.1 Final Component Organization

```
src/
├── components/
│   ├── generator/           # Generator-specific components
│   │   ├── context/
│   │   │   └── GeneratorFormContext.tsx
│   │   ├── ScenarioSelector.tsx
│   │   ├── FrameworkTemplateSection.tsx
│   │   ├── SigningInlineSection.tsx
│   │   ├── OutputLocationSelector.tsx
│   │   ├── ValidationErrors.tsx
│   │   └── index.ts
│   │
│   ├── layout/              # Layout components
│   │   ├── GeneratorLayout.tsx
│   │   ├── PipelineSidebar.tsx
│   │   ├── SidebarHeader.tsx
│   │   ├── SidebarNavigation.tsx
│   │   ├── DebugJsonModal.tsx
│   │   └── index.ts
│   │
│   ├── sections/            # Page sections
│   │   ├── configuration/
│   │   ├── bundle/
│   │   ├── preflight/
│   │   ├── generate/
│   │   ├── build/
│   │   ├── smoketest/
│   │   ├── distribution/
│   │   ├── shared/
│   │   └── index.ts
│   │
│   ├── preflight/           # Preflight-specific components
│   ├── signing/             # Signing-specific components
│   ├── distribution/        # Distribution-specific components
│   ├── scenario-inventory/  # Inventory-specific components
│   ├── state/               # State display components
│   ├── ui/                  # Base UI components (shadcn)
│   └── index.ts
│
├── domain/                  # Pure business logic
├── hooks/                   # Custom React hooks
├── lib/                     # Utilities
├── pages/                   # Page components
├── store/                   # Zustand stores
└── types/                   # TypeScript types
```

---

## Implementation Priority

### Must Have (Critical Fixes)
1. ✅ Fix DebugJsonModal crash
2. ✅ Fix double padding issue
3. ✅ Show pipeline status properly in sidebar

### Should Have (Core Improvements)
4. Break apart GeneratorForm into smaller components
5. Populate section content from GeneratorForm
6. Create GeneratorFormContext for shared state

### Nice to Have (Polish)
7. Remove unused code
8. Add progress indicators to sidebar
9. Improve section transitions/animations
10. Add keyboard navigation support

---

## Estimated Effort

| Phase | Effort | Risk |
|-------|--------|------|
| Phase 1 (Bug Fixes) | 2-4 hours | Low |
| Phase 2 (Padding) | 1-2 hours | Low |
| Phase 3 (Section Content) | 8-12 hours | Medium |
| Phase 4 (Sidebar UX) | 2-4 hours | Low |
| Phase 5 (Cleanup) | 1-2 hours | Low |
| Phase 6 (Architecture) | 4-6 hours | Medium |

**Total: ~18-30 hours**

---

## Questions Before Implementation

1. **Section Consolidation:** Should we consolidate some sections? For example:
   - Combine Bundle + Preflight into "Environment Setup"
   - Combine Build + Smoke Test into "Build & Verify"

2. **Form State Management:** Should we use react-hook-form for the generator form, or keep manual state management?

3. **Section Interactivity:** Should sections be independently actionable (run preflight without running bundle), or enforce linear progression?

4. **Breaking Changes:** Is it acceptable to restructure props passed to GeneratorPage from App.tsx?
