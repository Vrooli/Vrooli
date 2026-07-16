/**
 * Shared test utilities for component tests.
 *
 * Import the canonical render helper and fixtures from here so provider setup
 * (QueryClient, router, app context) stays centralized:
 *
 *   import { renderWithProviders, expectNoA11yViolations } from '../test-utils';
 */
export { expectNoA11yViolations } from './a11y';
export {
  renderWithProviders,
  type ProviderRenderOptions,
  type ProviderRenderResult,
} from './renderWithProviders';
