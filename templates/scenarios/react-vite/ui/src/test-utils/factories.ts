/**
 * Test data factories.
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `Partial<Domain>`. Defaults are picked
 * so the most common test path is `makeX()` with no args.
 *
 * Naming: `make<Domain>` (camelCase) — the TS analogue of workspace-
 * sandbox's Go-side `Fake<Domain>`. Asymmetry is deliberate: Go fakes
 * are stateful types (`type FakeClock struct{...}`); TS factories
 * return plain data (`HealthResponse`).
 *
 * When proto-types lands, the import below swaps to the generated
 * type and every callsite continues to work — `Partial<HealthResponse>`
 * narrows to the new shape automatically.
 */

// Local mirror of the wire shape returned by lib/api.fetchHealth.
// Kept as an interface (rather than `Awaited<ReturnType<typeof fetchHealth>>`)
// so factories don't pull lib/api into the test-utils dependency graph.
export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export const makeHealthResponse = (
  overrides: Partial<HealthResponse> = {},
): HealthResponse => ({
  status: "ok",
  service: "react-vite-test",
  timestamp: "2026-01-01T00:00:00.000Z",
  ...overrides,
});
