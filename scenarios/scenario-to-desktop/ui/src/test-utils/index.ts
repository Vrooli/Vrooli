/**
 * Test utilities barrel exports.
 * Re-exports commonly used testing utilities and custom helpers.
 */

// Custom render utilities
export {
  renderWithProviders,
  createHookWrapper,
  createTestQueryClient,
  type RenderWithProvidersOptions,
} from "./render";

// Mock factories
export {
  // Pipeline state mocks
  createPipelineState,
  createRunningPipelineState,
  createCompletedPipelineState,
  createFailedPipelineState,
  // Form state mocks
  createFormState,
  // API response mocks
  createPipelineStatus,
  createPreflightResponse,
  createPreflightWithMissingSecrets,
  createPreflightWithValidationErrors,
  // Stage result mocks
  createBundleResult,
  createGenerateResult,
  createBuildResult,
  createSmokeTestResult,
  createDistributionResult,
} from "./mocks";

// Re-export commonly used testing utilities from @testing-library/react
export { screen, fireEvent, waitFor, within, act } from "@testing-library/react";
export { renderHook } from "@testing-library/react";

// Re-export vitest utilities
export { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
