/**
 * Mock builders for `./api/health` — the UI ↔ API health boundary.
 *
 * # Why a builder, not a direct mock
 *
 * Vitest hoists `vi.mock(path, factory)` calls to before any user
 * imports run. A wrapper helper imported from `@/test-utils` would be
 * in the temporal dead zone at hoist time (we tried; the failure mode
 * is a TDZ error). The factory closure body, however, runs *after*
 * imports resolve, so calling `makeApiMocks()` from inside the closure
 * is safe — and lets the contract for the api/health stub live in one
 * file instead of being copy-pasted across every test that mocks it.
 *
 * # Canonical usage
 *
 *   import { makeApiMocks } from "@/test-utils";
 *
 *   vi.mock("./api/health", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("./api/health")>();
 *     return { ...actual, ...makeApiMocks() };
 *   });
 *
 * Per-test overrides (e.g. simulating a 5xx) use vitest's standard
 * pattern *after* the mock is wired:
 *
 *   const { fetchHealth } = await import("./api/health");
 *   vi.mocked(fetchHealth).mockRejectedValueOnce(new Error("boom"));
 *
 * Adding a new api/health function: extend `ApiMocks` and `makeApiMocks`
 * together. Keep `*` non-overridden via the `...actual` spread at the
 * call site so unrelated exports (constants, types, classes) keep
 * working.
 */
import { vi } from "vitest";

import { makeFixResponse, makeHealthResponse, makeValidationReport } from "../factories";

export interface ApiMocks {
  /** vi.fn() resolving to a healthy default; override via vi.mocked(...). */
  fetchHealth: ReturnType<typeof vi.fn>;
  /** vi.fn() resolving to a representative validation report. */
  validateScenario: ReturnType<typeof vi.fn>;
  /** vi.fn() resolving to a deterministic fix preview. */
  previewFix: ReturnType<typeof vi.fn>;
}

/**
 * Build a fresh `api/health` mock surface. Call from inside a `vi.mock`
 * factory closure — never at module top level (see file header).
 */
export const makeApiMocks = (): ApiMocks => ({
  fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  validateScenario: vi.fn().mockResolvedValue(makeValidationReport()),
  previewFix: vi.fn().mockResolvedValue(makeFixResponse()),
});
