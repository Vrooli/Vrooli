import { expect, describe, it } from "vitest";
import { groupPrismaData } from "./groupPrismaData.js";
import { type ModelType } from "@vrooli/shared";
import { type PartialApiInfo } from "./types.js";

describe("groupPrismaData", () => {
    describe("basic functionality", () => {
        it("returns empty dictionaries for invalid input", () => {
            expect(groupPrismaData(null as any, null as any)).toEqual({
                objectTypesIdsDict: {},
                selectFieldsDict: {},
                objectIdsDataDict: {},
            });

            expect(groupPrismaData(undefined as any, undefined as any)).toEqual({
                objectTypesIdsDict: {},
                selectFieldsDict: {},
                objectIdsDataDict: {},
            });
        });

        it("handles single object with ID", () => {
            const data = { id: "123", name: "Test" };
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result).toEqual({
                objectTypesIdsDict: { User: ["123"] },
                selectFieldsDict: {
                    User: {
                        id: true,
                        name: true,
                        __typename: "User",
                    },
                },
                objectIdsDataDict: {
                    "123": { id: "123", name: "Test" },
                },
            });
        });

        it("handles array of objects", () => {
            const data = [
                { id: "123", name: "User 1" },
                { id: "456", name: "User 2" },
            ];
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result).toEqual({
                objectTypesIdsDict: { User: ["123", "456"] },
                selectFieldsDict: {
                    User: {
                        id: true,
                        name: true,
                        __typename: "User",
                    },
                },
                objectIdsDataDict: {
                    "123": { id: "123", name: "User 1" },
                    "456": { id: "456", name: "User 2" },
                },
            });
        });

        it("removes duplicate IDs", () => {
            const data = [
                { id: "123", name: "User 1" },
                { id: "123", name: "User 1 Updated" },
                { id: "456", name: "User 2" },
            ];
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict.User).toEqual(["123", "456"]);
            // Later data should override earlier for same ID
            expect(result.objectIdsDataDict["123"]).toEqual({
                id: "123",
                name: "User 1 Updated",
            });
        });
    });

    describe("nested relationships", () => {
        it("handles single-level nesting", () => {
            const data = {
                id: "123",
                name: "Post 1",
                author: {
                    id: "456",
                    name: "Author 1",
                },
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "Post" as ModelType,
                author: {
                    id: true,
                    name: true,
                    __typename: "User" as ModelType,
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Post: ["123"],
                User: ["456"],
            });
            expect(result.selectFieldsDict).toHaveProperty("Post");
            expect(result.selectFieldsDict).toHaveProperty("User");
            expect(result.objectIdsDataDict["123"]).toEqual({
                id: "123",
                name: "Post 1",
                author: { id: "456", name: "Author 1" },
            });
            expect(result.objectIdsDataDict["456"]).toEqual({
                id: "456",
                name: "Author 1",
            });
        });

        it("handles arrays in nested relationships", () => {
            const data = {
                id: "123",
                title: "Team A",
                members: [
                    { id: "456", name: "Member 1" },
                    { id: "789", name: "Member 2" },
                ],
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                title: true,
                __typename: "Team" as ModelType,
                members: {
                    id: true,
                    name: true,
                    __typename: "User" as ModelType,
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Team: ["123"],
                User: ["456", "789"],
            });
            expect(result.objectIdsDataDict["456"]).toEqual({
                id: "456",
                name: "Member 1",
            });
            expect(result.objectIdsDataDict["789"]).toEqual({
                id: "789",
                name: "Member 2",
            });
        });

        it("handles deep nesting", () => {
            const data = {
                id: "1",
                name: "Project",
                team: {
                    id: "2",
                    name: "Team",
                    leader: {
                        id: "3",
                        name: "Leader",
                    },
                },
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "Project" as ModelType,
                team: {
                    id: true,
                    name: true,
                    __typename: "Team" as ModelType,
                    leader: {
                        id: true,
                        name: true,
                        __typename: "User" as ModelType,
                    },
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Project: ["1"],
                Team: ["2"],
                User: ["3"],
            });
        });
    });

    describe("union types", () => {
        it("handles union types based on __typename", () => {
            const data = {
                id: "123",
                title: "Bookmark",
                to: {
                    __typename: "User",
                    id: "456",
                    name: "Bookmarked User",
                },
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                title: true,
                __typename: "Bookmark" as ModelType,
                to: {
                    User: {
                        id: true,
                        name: true,
                        __typename: "User" as ModelType,
                    },
                    Team: {
                        id: true,
                        handle: true,
                        __typename: "Team" as ModelType,
                    },
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Bookmark: ["123"],
                User: ["456"],
            });
            expect(result.objectIdsDataDict["456"]).toEqual({
                __typename: "User",
                id: "456",
                name: "Bookmarked User",
            });
        });

        it("skips union when no matching type found", () => {
            const data = {
                id: "123",
                title: "Bookmark",
                to: {
                    // No __typename, so can't match union
                    id: "456",
                    name: "Unknown",
                },
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                title: true,
                __typename: "Bookmark" as ModelType,
                to: {
                    User: {
                        id: true,
                        name: true,
                        __typename: "User" as ModelType,
                    },
                    Team: {
                        id: true,
                        handle: true,
                        __typename: "Team" as ModelType,
                    },
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Bookmark: ["123"],
            });
            // Should not include the unmatched union data
            expect(result.objectIdsDataDict["456"]).toBeUndefined();
        });

        it("handles arrays of unions", () => {
            const data = {
                id: "123",
                title: "Multi Bookmark",
                items: [
                    {
                        __typename: "User",
                        id: "456",
                        name: "User 1",
                    },
                    {
                        __typename: "Team",
                        id: "789",
                        handle: "$team1",
                    },
                ],
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                title: true,
                __typename: "Bookmark" as ModelType,
                items: {
                    User: {
                        id: true,
                        name: true,
                        __typename: "User" as ModelType,
                    },
                    Team: {
                        id: true,
                        handle: true,
                        __typename: "Team" as ModelType,
                    },
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Bookmark: ["123"],
                User: ["456"],
                Team: ["789"],
            });
        });
    });

    describe("edge cases", () => {
        it("handles BigInt IDs", () => {
            const data = { id: BigInt("123456789012345678"), name: "Test" };
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict.User).toEqual(["123456789012345678"]);
            expect(result.objectIdsDataDict["123456789012345678"]).toBeDefined();
        });

        it("handles numeric IDs", () => {
            const data = { id: 123, name: "Test" };
            const partialInfo: PartialApiInfo = {
                id: true,
                name: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict.User).toEqual(["123"]);
        });

        it("ignores objects without IDs", () => {
            const data = { name: "No ID Object" };
            const partialInfo: PartialApiInfo = {
                name: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({});
            expect(result.objectIdsDataDict).toEqual({});
            expect(result.selectFieldsDict).toHaveProperty("User");
        });

        it("handles Date objects properly", () => {
            const testDate = new Date("2023-01-01");
            const data = {
                id: "123",
                createdAt: testDate,
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                createdAt: true,
                __typename: "User" as ModelType,
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectIdsDataDict["123"]).toEqual({
                id: "123",
                createdAt: testDate,
            });
        });

        it("handles mixed array of partialInfo", () => {
            const data = [
                { id: "123", name: "User" },
                { id: "456", handle: "$team" },
            ];
            const partialInfos = [
                { id: true, name: true, __typename: "User" as ModelType },
                { id: true, handle: true, __typename: "Team" as ModelType },
            ];

            const result = groupPrismaData(data, partialInfos);

            expect(result.objectTypesIdsDict).toEqual({
                User: ["123"],
                Team: ["456"],
            });
        });

        it("merges select fields when type appears multiple times", () => {
            const data = {
                id: "123",
                author: { id: "456", name: "Author" },
                reviewer: { id: "789", email: "reviewer@example.com" },
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                __typename: "Post" as ModelType,
                author: {
                    id: true,
                    name: true,
                    __typename: "User" as ModelType,
                },
                reviewer: {
                    id: true,
                    email: true,
                    __typename: "User" as ModelType,
                },
            };

            const result = groupPrismaData(data, partialInfo);

            // User select fields should be merged
            expect(result.selectFieldsDict.User).toEqual({
                id: true,
                name: true,
                email: true,
                __typename: "User",
            });
        });

        it("handles empty arrays", () => {
            const data = {
                id: "123",
                tags: [],
            };
            const partialInfo: PartialApiInfo = {
                id: true,
                __typename: "Post" as ModelType,
                tags: {
                    id: true,
                    name: true,
                    __typename: "Tag" as ModelType,
                },
            };

            const result = groupPrismaData(data, partialInfo);

            expect(result.objectTypesIdsDict).toEqual({
                Post: ["123"],
            });
        });
    });
});
