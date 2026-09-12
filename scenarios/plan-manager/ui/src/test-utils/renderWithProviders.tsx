/**
 * Scenario-owned render boundary for component tests.
 *
 * Keeping the app's i18n singleton and cross-cutting providers behind one
 * import makes provider changes deliberate and keeps isolated tests aligned
 * with the production composition.
 */
import {
  renderWithProviders as renderWithBaseProviders,
  type ProviderRenderOptions,
  type ProviderRenderResult,
} from "@vrooli/api-base/testing";
import type { ReactElement } from "react";

import { Providers } from "../app/providers";
import { i18n } from "../i18n";

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  return renderWithBaseProviders(ui, {
    i18n,
    extraProviders: (children) => <Providers>{children}</Providers>,
    ...options,
  });
}
