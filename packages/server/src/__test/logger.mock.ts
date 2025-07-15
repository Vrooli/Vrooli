/**
 * Mock logger for tests
 */
import { vi } from "vitest";

export const mockLogger = {
    info: vi.fn(),
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    verbose: vi.fn(),
    silly: vi.fn(),
};
