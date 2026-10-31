/** Scenario-owned canonical render boundary for component tests. */
import {
  renderWithProviders as renderWithBaseProviders,
  type ProviderRenderOptions,
  type ProviderRenderResult,
} from "@vrooli/api-base/testing";
import type { ReactElement } from "react";

import { i18n } from "../i18n";

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  return renderWithBaseProviders(ui, { i18n, ...options });
}
