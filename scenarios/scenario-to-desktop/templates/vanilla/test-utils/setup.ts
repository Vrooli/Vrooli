/**
 * Vitest Setup File
 *
 * DOC: docs/internal/SEAMS.md#test-setup
 *
 * Global setup for all tests. Runs before each test file.
 */

import { vi } from "vitest";

// Reset all mocks between tests
beforeEach(() => {
    vi.clearAllMocks();
});

// Clean up after all tests
afterAll(() => {
    vi.restoreAllMocks();
});
