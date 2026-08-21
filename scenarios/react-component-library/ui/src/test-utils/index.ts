/**
 * Test-only helpers shared across component and integration tests.
 *
 * Organized so consumers learn one import path and discover capabilities
 * via auto-complete. Production code MUST NOT import from this directory
 * — `eslint.config.js` enforces this via `no-restricted-imports`
 * (mirroring the Go-side AST guardrail at
 * `api/internal/testutil/no_prod_import_test.go`). The override block
 * for `*.test.{ts,tsx}` / `*.spec.{ts,tsx}` turns the rule off so
 * tests import from here freely.
 *
 * # What lives here
 *
 *   - `renderWithProviders` — render an element wrapped in QueryClient
 *     and i18n providers. Use this instead of bare `render()` so future
 *     provider additions (router, theme) propagate to every test.
 *   - `make<Domain>` factories — stable typed default test data with
 *     `Partial<Domain>` overrides.
 *   - `mocks/` — shared mock builders for external SDKs (e.g. the
 *     spatial-nav module). Inline `vi.mock(...)` factories invoke
 *     these so the contract for each SDK lives in one file.
 *
 * # API mocking is intentionally NOT a helper
 *
 * Vitest hoists `vi.mock(path, factory)` calls to the very top of the
 * file — before any user imports run. A wrapper helper imported from
 * here would itself be in the temporal dead zone when hoisted code
 * tries to call it (we tried; the failure mode is a TDZ error). The
 * canonical pattern is therefore inline at each test:
 *
 *   import { makeHealthResponse } from "@/test-utils";
 *
 *   vi.mock("./api/health", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("./api/health")>();
 *     return {
 *       ...actual,
 *       fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
 *     };
 *   });
 *
 *   import App from "./App";
 *
 * `makeHealthResponse()` is referenced from inside the factory closure,
 * which runs when vitest resolves the mock — *after* imports are
 * initialised. The pattern above is hoisting-safe and preserves every
 * non-overridden export of `./api/health` via `importOriginal()`.
 */
export { expectNoA11yViolations } from "@vrooli/api-base/testing";
export type { ProviderRenderOptions, ProviderRenderResult } from "@vrooli/api-base/testing";
export { renderWithProviders } from "./renderWithProviders";
export { interp } from "./interp";
// Note: HealthResponse is the *generated proto type* re-exported by
// factories.ts. Tests should always import it from here so a future
// schema change is one-import-update; consuming the proto package
// directly in tests fragments that contract.
//
// Domain-specific factories (Note, NotesListResponse, etc.) are NOT
// re-exported here — they live next to the feature they double for
// (e.g. `features/notes/mocks/factories.ts`) so deleting a feature
// folder takes them along.
export { makeHealthResponse } from "./factories";
export type { HealthResponse } from "./factories";
export { assertTransitionMatrix, validateTransitionMatrix } from "./modeltest/matrix";
export type { MatrixRow, WorkflowTransition } from "./modeltest/matrix";
export { replayTraces, validateTraces } from "./modeltest/traces";
export type { Trace, TraceStep } from "./modeltest/traces";
export {
  assertFormalArtifactFresh,
  assertFormalTransitionsReplay,
  assertFormalTracesReplay,
  transitionFromReplayAdapter,
  validateFormalArtifactFresh,
  validateFormalTransitionsReplay,
  validateFormalTracesReplay,
} from "./modeltest/formal";
export type {
  FormalArtifact,
  FormalArtifactTraceCoverage,
  FormalReplayAdapter,
  FormalArtifactTrace,
  FormalArtifactTraceStep,
  FormalArtifactTransition,
} from "./modeltest/formal";

// Mock builders for external SDKs. Each test file still calls
// `vi.mock(<module>, ...)` inline (Vitest hoisting requires it); the
// builders live in one place so a future API addition is a one-edit
// change rather than a fan-out across consumers.
export {
  makeGamepadInputManagerCtor,
  makeMockGamepadInputManager,
  makeMockSpatialNavController,
} from "./mocks/spatial";
export type { MockGamepadInputManager, MockSpatialNavController } from "./mocks/spatial";

// Internal-seam mock builders for cross-domain HTTP wrappers (the
// generic `api/health` health/error path). Domain-specific mocks
// (e.g. `makeNotesMocks`) live with their feature.
// Use `...makeApiMocks()` *inside* the
// `vi.mock(..., async (importOriginal) => …)` factory closure — never
// at module top level. See mocks/api.ts for the canonical usage shape.
export { makeApiMocks } from "./mocks/api";
export type { ApiMocks } from "./mocks/api";
