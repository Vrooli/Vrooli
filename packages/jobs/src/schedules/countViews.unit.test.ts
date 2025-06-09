import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock all external dependencies to create a pure unit test
const mockBatch = vi.fn();
const mockLogger = {
    info: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
};
const mockDbProvider = {
    get: vi.fn(() => ({
        $transaction: vi.fn(),
        user: {
            findMany: vi.fn(),
            updateMany: vi.fn(),
        },
        team: {
            findMany: vi.fn(),
            updateMany: vi.fn(),
        },
        resource: {
            findMany: vi.fn(),
            updateMany: vi.fn(),
        },
        issue: {
            findMany: vi.fn(),
            updateMany: vi.fn(),
        },
    })),
};

// Mock all imports
vi.mock("@vrooli/server/db/provider.js", () => ({
    DbProvider: mockDbProvider,
}));

vi.mock("@vrooli/server/utils/batch.js", () => ({
    batch: mockBatch,
}));

vi.mock("@vrooli/server/events/logger.js", () => ({
    logger: mockLogger,
}));

// Import after mocking
const { countViews } = await import("./countViews.js");

describe("countViews unit tests", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should call batch for each entity type", async () => {
        // Setup mock to avoid actual database calls
        mockBatch.mockImplementation(async ({ processBatch }) => {
            // Simulate empty result
            await processBatch([]);
        });

        await countViews();

        // Verify batch was called for each entity type
        expect(mockBatch).toHaveBeenCalledTimes(4);
        
        // Check the entity types are correct
        const batchCalls = mockBatch.mock.calls;
        const entityTypes = batchCalls.map(call => call[0].objectType);
        expect(entityTypes).toEqual(['User', 'Team', 'Resource', 'Issue']);
    });

    it("should handle batch processing with mismatched counts", async () => {
        const mockDb = mockDbProvider.get();
        
        // Mock batch to provide test data
        mockBatch.mockImplementation(async ({ processBatch, objectType }) => {
            if (objectType === 'User') {
                const mockUsers = [
                    { id: 1n, views: 5 }, // Wrong count
                    { id: 2n, views: 0 }, // Correct count
                ];
                await processBatch(mockUsers);
            } else {
                await processBatch([]);
            }
        });

        // Mock database queries
        mockDb.user.findMany.mockResolvedValue([
            { userId: 1n, count: 3 }, // Actual count is 3, not 5
            { userId: 2n, count: 0 }, // Correct
        ]);

        await countViews();

        // Should only update the mismatched user
        expect(mockDb.user.updateMany).toHaveBeenCalledWith({
            where: { id: { in: [1n] } },
            data: { views: 3 },
        });
    });

    it("should log information about processing", async () => {
        mockBatch.mockImplementation(async ({ processBatch }) => {
            await processBatch([]);
        });

        await countViews();

        expect(mockLogger.info).toHaveBeenCalledWith("Starting view count verification and correction...");
        expect(mockLogger.info).toHaveBeenCalledWith("View count verification complete.");
    });

    it("should handle errors gracefully", async () => {
        const error = new Error("Database error");
        mockBatch.mockRejectedValue(error);

        await expect(countViews()).rejects.toThrow("Database error");
    });

    it("should update entities when view counts are zero but database has non-zero", async () => {
        const mockDb = mockDbProvider.get();
        
        mockBatch.mockImplementation(async ({ processBatch, objectType }) => {
            if (objectType === 'Issue') {
                // Entity has views=10 but should be 0
                await processBatch([{ id: 1n, views: 10 }]);
            } else {
                await processBatch([]);
            }
        });

        // Mock no actual views for this issue
        mockDb.issue.findMany.mockResolvedValue([]);

        await countViews();

        // Should update to 0
        expect(mockDb.issue.updateMany).toHaveBeenCalledWith({
            where: { id: { in: [1n] } },
            data: { views: 0 },
        });
    });
});