/**
 * Test Utilities Index
 *
 * DOC: docs/internal/SEAMS.md#test-utilities
 *
 * Exports shared test utilities and mock factories.
 * Import from this file in test files:
 *
 * ```typescript
 * import { createMockApp, createMockBrowserWindow, createMockFs } from "../test-utils";
 * ```
 */

// Re-export all Electron mocks
export * from "./electron-mocks";

// Re-export filesystem mocks
export * from "./fs-mocks";

// Re-export async helpers
export * from "./async-helpers";
