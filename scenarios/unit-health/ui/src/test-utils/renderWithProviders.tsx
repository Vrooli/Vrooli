/**
 * unit-health's render helper.
 *
 * It wraps the shared `@vrooli/api-base/testing` helper rather than replacing
 * it, and adds the one thing api-base cannot own: this scenario's
 * `ThemeProvider`. Theme is a scenario-local provider — api-base has no
 * business knowing that unit-health resolves `system` through a media query and
 * writes `data-theme` onto `<html>` — so the seam api-base exposes for it is
 * `extraProviders`, and this file is where that seam is bound.
 *
 * This file exists because re-exporting the shared helper unchanged is not
 * enough: `TopBar` and `SettingsPage` both call `useTheme()`, which throws
 * outside a provider. Every test that renders the shell (`AppShell.test.tsx`,
 * `AppShell.a11y.test.tsx`, `app/routes.test.tsx`) therefore died on
 * "useTheme must be called inside <ThemeProvider>" the moment the barrel
 * pointed straight at api-base.
 *
 * It also binds this scenario's i18n singleton. api-base defaults to an
 * instance it creates itself, pinned to `cimode` with empty resources. That is
 * invisible in key-echo tests — cimode renders the key path either way — but it
 * silently breaks every test that asserts real translated copy, because the
 * scenario's locale bundles were never loaded into the instance doing the
 * rendering. Passing the singleton makes `setLocale("en")` in a test actually
 * reach the tree it renders.
 *
 * `initialTheme` is offered because the alternative is a test that passes for
 * the wrong reason. With no seeded choice the provider resolves `system`
 * through jsdom's absent `matchMedia`, which happens to answer "light" today
 * and would answer differently the moment jsdom stubs it. Leave it unset to
 * exercise the real resolution path; set it when the assertion is about a
 * specific theme rather than about the harness.
 */
import type { ReactElement, ReactNode } from "react";

import {
  renderWithProviders as renderWithSharedProviders,
  type ProviderRenderOptions as SharedProviderRenderOptions,
  type ProviderRenderResult,
} from "@vrooli/api-base/testing";

import { i18n as scenarioI18n } from "../i18n";
import { ThemeProvider, type ThemeChoice } from "../theme/ThemeProvider";

export interface ProviderRenderOptions extends SharedProviderRenderOptions {
  /**
   * Seeds `ThemeProvider`'s choice, bypassing localStorage and the media
   * query. Omit it to exercise the real resolution path.
   */
  initialTheme?: ThemeChoice;
}

export type { ProviderRenderResult };

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  const { initialTheme, extraProviders, i18n = scenarioI18n, ...shared } = options;
  const wrapScenarioProviders = (children: ReactNode): ReactNode => (
    <ThemeProvider initialChoice={initialTheme}>
      {extraProviders ? extraProviders(children) : children}
    </ThemeProvider>
  );
  return renderWithSharedProviders(ui, {
    ...shared,
    i18n,
    extraProviders: wrapScenarioProviders,
  });
}
