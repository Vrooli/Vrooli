/**
 * Shared test render entry point for the Test Genie UI.
 *
 * Keeping this projection beside the UI tests gives the unit-health contract
 * one stable import surface while delegating provider wiring to api-base.
 */
export {
  renderWithProviders,
} from "@vrooli/api-base/testing";

export type {
  ProviderRenderOptions,
  ProviderRenderResult,
} from "@vrooli/api-base/testing";
