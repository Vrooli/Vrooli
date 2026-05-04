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
 *   vi.mock("./lib/api", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("./lib/api")>();
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
 * non-overridden export of `./lib/api` via `importOriginal()`.
 */
export { renderWithProviders } from "./renderWithProviders";
export type { ProviderRenderOptions, ProviderRenderResult } from "./renderWithProviders";
export { interp } from "./interp";
// Note: HealthResponse is the *generated proto type* re-exported by
// factories.ts. Tests should always import it from here so a future
// schema change is one-import-update; consuming the proto package
// directly in tests fragments that contract.
export { makeHealthResponse, makeListNotesResponse, makeNote } from "./factories";
export type { HealthResponse, ListNotesResponse, Note } from "./factories";

// Mock builders for external SDKs and internal seams. Each test file
// still calls `vi.mock(<module>, ...)` inline (Vitest hoisting requires
// it); the builders live in one place so a future API addition is a
// one-edit change rather than a fan-out across consumers.
export {
  makeGamepadInputManagerCtor,
  makeMockGamepadInputManager,
  makeMockSpatialNavController,
} from "./mocks/spatial";
export type {
  MockGamepadInputManager,
  MockSpatialNavController,
} from "./mocks/spatial";

// Internal-seam mock builders for the UI's own lib/* HTTP wrappers.
// Use `...makeApiMocks()` / `...makeNotesMocks()` *inside* the
// `vi.mock(..., async (importOriginal) => …)` factory closure — never
// at module top level. See mocks/api.ts and mocks/notes.ts for the
// canonical usage shape.
export { makeApiMocks } from "./mocks/api";
export type { ApiMocks } from "./mocks/api";
export { makeNotesMocks } from "./mocks/notes";
export type { NotesMocks, NotesMockCreateInput } from "./mocks/notes";
