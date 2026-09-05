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
