// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 7 'as any' type assertions with proper ProjectVersionDirectory type
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { initializeDirectoryList } from "./DirectoryList.js";
import type { ProjectVersionDirectory } from "@vrooli/shared";

type SimpleVersionObject = {
    __typename: string;
    id: string;
    createdAt: string;
    updatedAt: string;
    translations: Array<{
        __typename: string;
        id: string;
        language: string;
        name: string;
    }>;
}

// Mock data for testing
const mockSession = {
    __typename: "Session" as const,
    isLoggedIn: true,
};
// Mock directory with proper typed structure
const mockDirectory: Partial<ProjectVersionDirectory> = {
    childApiVersions: [{
        __typename: "ApiVersion",
        id: "api-a-id",
        createdAt: "2023-12-22T01:52:51.304Z",
        updatedAt: "2023-12-23T05:00:00.000Z",
        translations: [{ __typename: "ApiVersionTranslation", id: "trans-1", language: "en", name: "API A" }]
    }],
    childCodeVersions: [],
    childNoteVersions: [{
        __typename: "NoteVersion",
        id: "note-b-id",
        createdAt: "2023-12-20T01:52:51.304Z",
        updatedAt: "2023-12-21T05:00:00.000Z",
        translations: [{ __typename: "NoteVersionTranslation", id: "trans-2", language: "en", name: "Note B" }]
    }],
    childProjectVersions: [{
        __typename: "ProjectVersion",
        id: "project-c-id",
        createdAt: "2023-12-21T01:52:51.304Z",
        updatedAt: "2023-12-22T05:00:00.000Z",
        translations: [{ __typename: "ProjectVersionTranslation", id: "trans-3", language: "en", name: "Project C" }]
    }],
    childRoutineVersions: [],
    childStandardVersions: [],
    childTeams: [],
};

describe("initializeDirectoryList", () => {
    it("returns an empty array when directory is null", () => {
        const result = initializeDirectoryList(null, "NameAsc", "Hourly", mockSession);
        expect(result).toHaveLength(0);
    });

    it("sorts by NameAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as ProjectVersionDirectory, "NameAsc", "Daily", mockSession);
        // Note: results are sorted alphabetically, so we check the translations for display names
        expect(result).toHaveLength(3);
        expect(result[0].translations?.[0]?.name).toBe("API A");
        expect(result[1].translations?.[0]?.name).toBe("Note B");
        expect(result[2].translations?.[0]?.name).toBe("Project C");
    });

    it("sorts by NameDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as ProjectVersionDirectory, "NameDesc", "Weekly", mockSession);
        expect(result).toHaveLength(3);
        expect(result[0].translations?.[0]?.name).toBe("Project C");
        expect(result[1].translations?.[0]?.name).toBe("Note B");
        expect(result[2].translations?.[0]?.name).toBe("API A");
    });

    it("sorts by DateCreatedAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as ProjectVersionDirectory, "DateCreatedAsc", "AllTime", mockSession);
        expect(result).toHaveLength(3);
        expect(result[0].translations?.[0]?.name).toBe("Note B");
        expect(result[1].translations?.[0]?.name).toBe("Project C");
        expect(result[2].translations?.[0]?.name).toBe("API A");
    });

    it("sorts by DateCreatedDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as ProjectVersionDirectory, "DateCreatedDesc", "Hourly", mockSession);
        expect(result).toHaveLength(3);
        expect(result[0].translations?.[0]?.name).toBe("API A");
        expect(result[1].translations?.[0]?.name).toBe("Project C");
        expect(result[2].translations?.[0]?.name).toBe("Note B");
    });

    it("sorts by DateUpdatedAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as ProjectVersionDirectory, "DateUpdatedAsc", "Weekly", mockSession);
        expect(result).toHaveLength(3);
        expect(result[0].translations?.[0]?.name).toBe("Note B");
        expect(result[1].translations?.[0]?.name).toBe("Project C");
        expect(result[2].translations?.[0]?.name).toBe("API A");
    });

    it("sorts by DateUpdatedDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as ProjectVersionDirectory, "DateUpdatedDesc", "Monthly", mockSession);
        expect(result).toHaveLength(3);
        expect(result[0].translations?.[0]?.name).toBe("API A");
        expect(result[1].translations?.[0]?.name).toBe("Project C");
        expect(result[2].translations?.[0]?.name).toBe("Note B");
    });

    it("handles invalid dates gracefully", () => {
        const invalidDateDirectory: Partial<ProjectVersionDirectory> = {
            ...mockDirectory,
            childApiVersions: mockDirectory.childApiVersions ? [{
                ...mockDirectory.childApiVersions[0],
                createdAt: "invalid-date"
            }] : [],
        };
        const result = initializeDirectoryList(invalidDateDirectory as ProjectVersionDirectory, "DateCreatedAsc", "Daily", mockSession);
        // The item with the invalid date should be treated as if it has the earliest possible date
        expect(result[0].translations?.[0]?.name).toBe("API A");
    });
});
