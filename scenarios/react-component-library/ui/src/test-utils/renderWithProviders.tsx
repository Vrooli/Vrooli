/**
 * Scenario-bound render helper.
 *
 * `@vrooli/api-base/testing` ships a generic `renderWithProviders` whose
 * `I18nextProvider` falls back to a package-private i18next instance
 * pinned to `cimode` with **empty resources**. That default is correct for
 * scenarios that have no i18n of their own, but this scenario does: every
 * component resolves copy through the singleton created in `src/i18n`.
 *
 * Rendering against the package-private instance made every test render key
 * paths no matter what language the test had selected, so any test that opted
 * into a real locale (`await setLocale("en")`) still saw `health.refreshCount`
 * instead of "Refreshed once" — plural forms, `getByRole({ name })` lookups
 * against real labels, and locale-driven copy could never pass.
 *
 * This wrapper binds the render tree to the scenario singleton — the same
 * instance `App.tsx` uses and the same one `test-setup.ts` puts into `cimode`
 * before each test — so the language a test selects is the language it renders.
 * Callers can still override with an explicit `i18n` option.
 */
import {
  renderWithProviders as renderWithBaseProviders,
  type ProviderRenderOptions,
  type ProviderRenderResult,
} from "@vrooli/api-base/testing";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement } from "react";

import { i18n } from "../i18n";

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  const {
    routerEntries,
    initialEntries,
    initialIndex,
    route,
    withoutRouter,
    withRouter,
    ...providerOptions
  } = options;
  const entries = routerEntries ?? initialEntries ?? (route ? [route] : ["/"]);
  const shouldUseRouter = withRouter === undefined ? !withoutRouter : withRouter;
  const routed = shouldUseRouter ? (
    <MemoryRouter
      initialEntries={entries}
      initialIndex={initialIndex}
    >
      {ui}
    </MemoryRouter>
  ) : ui;

  return renderWithBaseProviders(routed, {
    i18n,
    withoutRouter: true,
    ...providerOptions,
  });
}
