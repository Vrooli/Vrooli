// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { initializeDirectoryList } from "./DirectoryList.js";

type SimpleObject = {
    createdAt: string;
    updatedAt: string;
    title: string;
}

// Mock data for testing
const mockSession = {
    __typename: "Session" as const,
    isLoggedIn: true,
};
// For simplicity, we'll use simplified versions of the types
const mockDirectory: {
    childApiVersions: SimpleObject[];
    childCodeVersions: SimpleObject[];
    childNoteVersions: SimpleObject[];
    childProjectVersions: SimpleObject[];
    childRoutineVersions: SimpleObject[];
    childStandardVersions: SimpleObject[];
    childTeams: SimpleObject[];
} = {
    childApiVersions: [{ createdAt: "2023-12-22T01:52:51.304Z", updatedAt: "2023-12-23T05:00:00.000Z", title: "API A" }],
    childCodeVersions: [],
    childNoteVersions: [{ createdAt: "2023-12-20T01:52:51.304Z", updatedAt: "2023-12-21T05:00:00.000Z", title: "Note B" }],
    childProjectVersions: [{ createdAt: "2023-12-21T01:52:51.304Z", updatedAt: "2023-12-22T05:00:00.000Z", title: "Project C" }],
    childRoutineVersions: [],
    childStandardVersions: [],
    childTeams: [],
};

describe("initializeDirectoryList", () => {
    it("returns an empty array when directory is null", () => {
        const result = initializeDirectoryList(null, "NameAsc", "Hourly", mockSession) as unknown as SimpleObject[];
        expect(result).toHaveLength(0);
    });

    it("sorts by NameAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "NameAsc", "Daily", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("API A");
        expect(result[1].title).toBe("Note B");
        expect(result[2].title).toBe("Project C");
    });

    it("sorts by NameDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "NameDesc", "Weekly", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("Project C");
        expect(result[1].title).toBe("Note B");
        expect(result[2].title).toBe("API A");
    });

    it("sorts by DateCreatedAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateCreatedAsc", "AllTime", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("Note B");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("API A");
    });

    it("sorts by DateCreatedDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateCreatedDesc", "Hourly", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("API A");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("Note B");
    });

    it("sorts by DateUpdatedAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateUpdatedAsc", "Weekly", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("Note B");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("API A");
    });

    it("sorts by DateUpdatedDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateUpdatedDesc", "Monthly", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("API A");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("Note B");
    });

    it("handles invalid dates gracefully", () => {
        const invalidDateDirectory = {
            ...mockDirectory,
            childApiVersions: [{ ...mockDirectory.childApiVersions[0], createdAt: "invalid-date" }],
        };
        const result = initializeDirectoryList(invalidDateDirectory as any, "DateCreatedAsc", "Daily", mockSession) as unknown as SimpleObject[];
        // The item with the invalid date should be treated as if it has the earliest possible date
        expect(result[0].title).toBe("API A");
    });
});
