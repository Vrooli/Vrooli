/**
 * Canonical render seam for authenticator UI tests.
 *
 * Keep the scenario-owned path stable for Test Genie and route all provider
 * setup through the shared API-base helper so new providers propagate to the
 * suite without direct Testing Library renders.
 */
export {
  renderWithProviders,
} from "@vrooli/api-base/testing";
export type {
  ProviderRenderOptions,
  ProviderRenderResult,
} from "@vrooli/api-base/testing";
