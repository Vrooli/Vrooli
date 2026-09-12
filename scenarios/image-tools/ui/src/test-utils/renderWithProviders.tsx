/**
 * image-tools' render helper.
 *
 * It wraps the shared `@vrooli/api-base/testing` helper rather than replacing
 * it, and adds exactly one thing api-base cannot own: this scenario's
 * `ThemeProvider`. Theme is a scenario-local provider — api-base has no
 * business knowing that image-tools resolves `system` through a media query and
 * writes `data-theme` onto `<html>` — so the seam api-base exposes for it is
 * `extraProviders`, and this file is where that seam is bound.
 *
 * `initialTheme` matters because the alternative is a test that passes for the
 * wrong reason. `TopBar.test.tsx` asserts the theme select reads "light"; with
 * no seeded choice the provider resolves `system` through jsdom's absent
 * `matchMedia`, which happens to answer "light" today and would answer
 * differently the moment jsdom stubs it. Seeding the choice makes the
 * assertion about the component instead of about the harness.
 */
import type { ReactElement, ReactNode } from "react";

import {
  renderWithProviders as renderWithSharedProviders,
  type ProviderRenderOptions as SharedProviderRenderOptions,
  type ProviderRenderResult,
} from "@vrooli/api-base/testing";

import { SettingsProvider } from "../features/settings/SettingsProvider";
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
  const { initialTheme, extraProviders, ...shared } = options;
  const wrapScenarioProviders = (children: ReactNode): ReactNode => (
    <ThemeProvider initialChoice={initialTheme}>
      <SettingsProvider>{extraProviders ? extraProviders(children) : children}</SettingsProvider>
    </ThemeProvider>
  );
  return renderWithSharedProviders(ui, { ...shared, extraProviders: wrapScenarioProviders });
}
