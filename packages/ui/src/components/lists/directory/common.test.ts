import { initializeDirectoryList } from "./common"; // Update with the actual path

type SimpleObject = {
    created_at: string;
    updated_at: string;
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
    childNoteVersions: SimpleObject[];
    childOrganizations: SimpleObject[];
    childProjectVersions: SimpleObject[];
    childRoutineVersions: SimpleObject[];
    childSmartContractVersions: SimpleObject[];
    childStandardVersions: SimpleObject[];
} = {
    childApiVersions: [{ created_at: "2023-12-22T01:52:51.304Z", updated_at: "2023-12-23T05:00:00.000Z", title: "API A" }],
    childNoteVersions: [{ created_at: "2023-12-20T01:52:51.304Z", updated_at: "2023-12-21T05:00:00.000Z", title: "Note B" }],
    childOrganizations: [],
    childProjectVersions: [{ created_at: "2023-12-21T01:52:51.304Z", updated_at: "2023-12-22T05:00:00.000Z", title: "Project C" }],
    childRoutineVersions: [],
    childSmartContractVersions: [],
    childStandardVersions: [],
};

describe("initializeDirectoryList", () => {
    it("returns an empty array when directory is null", () => {
        const result = initializeDirectoryList(null, "NameAsc", mockSession) as unknown as SimpleObject[];
        expect(result).toEqual([]);
    });

    it("sorts by NameAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "NameAsc", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("API A");
        expect(result[1].title).toBe("Note B");
        expect(result[2].title).toBe("Project C");
    });

    it("sorts by NameDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "NameDesc", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("Project C");
        expect(result[1].title).toBe("Note B");
        expect(result[2].title).toBe("API A");
    });

    it("sorts by DateCreatedAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateCreatedAsc", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("Note B");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("API A");
    });

    it("sorts by DateCreatedDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateCreatedDesc", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("API A");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("Note B");
    });

    it("sorts by DateUpdatedAsc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateUpdatedAsc", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("Note B");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("API A");
    });

    it("sorts by DateUpdatedDesc correctly", () => {
        const result = initializeDirectoryList(mockDirectory as any, "DateUpdatedDesc", mockSession) as unknown as SimpleObject[];
        expect(result[0].title).toBe("API A");
        expect(result[1].title).toBe("Project C");
        expect(result[2].title).toBe("Note B");
    });

    it("handles invalid dates gracefully", () => {
        const invalidDateDirectory = {
            ...mockDirectory,
            childApiVersions: [{ ...mockDirectory.childApiVersions[0], created_at: "invalid-date" }],
        };
        const result = initializeDirectoryList(invalidDateDirectory as any, "DateCreatedAsc", mockSession) as unknown as SimpleObject[];
        // The item with the invalid date should be treated as if it has the earliest possible date
        expect(result[0].title).toBe("API A");
    });
});
