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
 *   - `chooseOption(label, option)` — pick an option the way a person does,
 *     for a native select or a library Select.
 *
 * # Requests are observed at the wire
 *
 * Every API call goes through one Connect transport that looks up
 * `globalThis.fetch` per call, so a test stubs `fetch` and asserts the exact
 * URL and JSON body the desk would send. No service module is mocked.
 */
import { createElement, type ReactElement, type ReactNode } from "react";
import { renderWithProviders as renderWithBaseProviders, type ProviderRenderOptions, type ProviderRenderResult } from "@vrooli/api-base/testing";
import { i18n } from "../i18n";
import { ThemeProvider } from "../theme/ThemeProvider";

/**
 * Renders with this scenario's providers: the shared api-base wrapper (query
 * client, router, i18n provider) fed this scenario's initialized i18n
 * instance, plus the ThemeProvider every page expects. Without the `i18n`
 * argument the base wrapper falls back to an empty `cimode` instance and every
 * "real locale" assertion fails.
 */
export function renderWithProviders(ui: ReactElement, options: ProviderRenderOptions = {}): ProviderRenderResult {
  const extraProviders = options.extraProviders ?? ((children: ReactNode) => createElement(ThemeProvider, null, children));
  return renderWithBaseProviders(ui, { ...options, i18n, extraProviders });
}
export type { ProviderRenderOptions, ProviderRenderResult } from "@vrooli/api-base/testing";
export { interp } from "./interp";
export { expectNoA11yViolations } from "@vrooli/api-base/testing";
// Choose an option in a select by its label, for a native <select> or a
// library Select (whose trigger, not its hidden native select, is the control).
export { chooseOption } from "@vrooli/api-base/testing";
export {
  assertTransitionMatrix,
  validateTransitionMatrix,
} from "@vrooli/flow-runtime";
export type {
  MatrixRow,
  WorkflowTransition,
} from "@vrooli/flow-runtime";
export {
  replayTraces,
  validateTraces,
} from "@vrooli/flow-runtime";
export type {
  Trace,
  TraceStep,
} from "@vrooli/flow-runtime";
export {
  assertFormalArtifactFresh,
  assertFormalTransitionsReplay,
  assertFormalTracesReplay,
  transitionFromReplayAdapter,
  validateFormalArtifactFresh,
  validateFormalTransitionsReplay,
  validateFormalTracesReplay,
} from "@vrooli/flow-runtime";
export type {
  FormalArtifact,
  FormalArtifactTraceCoverage,
  FormalReplayAdapter,
  FormalArtifactTrace,
  FormalArtifactTraceStep,
  FormalArtifactTransition,
} from "@vrooli/flow-runtime";
export { stubWire, WireError, type WireCall } from "./wire";
