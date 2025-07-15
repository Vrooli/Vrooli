import { vi, beforeEach } from "vitest";

// Mock @vrooli/shared package to prevent translation loading issues during coverage
vi.mock("@vrooli/shared", async () => {
    const actualShared = await vi.importActual("@vrooli/shared");
    return {
        ...actualShared,
        // Mock any problematic imports here if needed
    };
});

// Global test setup
beforeEach(() => {
    // Reset console methods to avoid interference between tests
    vi.clearAllMocks();
});
