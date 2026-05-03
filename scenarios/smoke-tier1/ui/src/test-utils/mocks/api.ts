/**
 * Mock builders for `./lib/api` — the UI ↔ API HTTP boundary.
 *
 * # Why a builder, not a direct mock
 *
 * Vitest hoists `vi.mock(path, factory)` calls to before any user
 * imports run. A wrapper helper imported from `@/test-utils` would be
 * in the temporal dead zone at hoist time (we tried; the failure mode
 * is a TDZ error). The factory closure body, however, runs *after*
 * imports resolve, so calling `makeApiMocks()` from inside the closure
 * is safe — and lets the contract for the lib/api stub live in one
 * file instead of being copy-pasted across every test that mocks it.
 *
 * # Canonical usage
 *
 *   import { makeApiMocks } from "@/test-utils";
 *
 *   vi.mock("./lib/api", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("./lib/api")>();
 *     return { ...actual, ...makeApiMocks() };
 *   });
 *
 * Per-test overrides (e.g. simulating a 5xx) use vitest's standard
 * pattern *after* the mock is wired:
 *
 *   const { fetchHealth } = await import("./lib/api");
 *   vi.mocked(fetchHealth).mockRejectedValueOnce(new Error("boom"));
 *
 * Adding a new lib/api function: extend `ApiMocks` and `makeApiMocks`
 * together. Keep `*` non-overridden via the `...actual` spread at the
 * call site so unrelated exports (constants, types, classes) keep
 * working.
 */
import { vi } from "vitest";

import { makeHealthResponse } from "../factories";

export interface ApiMocks {
  /** vi.fn() resolving to a healthy default; override via vi.mocked(...). */
  fetchHealth: ReturnType<typeof vi.fn>;
}

/**
 * Build a fresh `lib/api` mock surface. Call from inside a `vi.mock`
 * factory closure — never at module top level (see file header).
 */
export const makeApiMocks = (): ApiMocks => ({
  fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
});
