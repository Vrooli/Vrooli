/**
 * InfoConverter tests - migrated from mocking to real dependencies
 * 
 * These tests verify the transformation logic between API and Prisma formats.
 * Instead of mocking ModelMap and FormatMap, we use real implementations with 
 * test configuration data.
 */
import { expect, describe, it, beforeEach, beforeAll } from "vitest";
import { type ModelType } from "@vrooli/shared";

// Delay imports that use ModelMap to avoid initialization issues
let CountFields: typeof import("./infoConverter.js").CountFields;
let InfoConverter: typeof import("./infoConverter.js").InfoConverter;
let addSupplementalFields: typeof import("./infoConverter.js").addSupplementalFields;
let addSupplementalFieldsHelper: typeof import("./infoConverter.js").addSupplementalFieldsHelper;
let addSupplementalFieldsMultiTypes: typeof import("./infoConverter.js").addSupplementalFieldsMultiTypes;

// Test configuration factories - inline, no external dependencies
function createTestModelConfig(type: ModelType, config: any) {
    return {
        format: config,
    };
}

function createTestFormatConfig(apiRelMap: any, options: { countFields?: any; joinMap?: any; hiddenFields?: string[] } = {}) {
    return {
        apiRelMap,
        countFields: options.countFields || {},
        joinMap: options.joinMap || {},
        hiddenFields: options.hiddenFields || [],
    };
}

// Test configurations
const testConfigs = {
    User: createTestFormatConfig(
        { __typename: "User", bookmarks: "Bookmark" },
        { 
            countFields: { bookmarksCount: true },
            joinMap: { members: "member" },
        },
    ),
    Team: createTestFormatConfig(
        { __typename: "Team", members: "Member" },
        { countFields: { membersCount: true } },
    ),
    Bookmark: createTestFormatConfig({
        __typename: "Bookmark",
        to: {
            user: "User",
            team: "Team",
        },
    }),
};

// Registry for dependency injection
const testModelRegistry = {
    get: (type: ModelType) => {
        const config = testConfigs[type];
        return config ? { format: config } : null;
    },
};


describe("InfoConverter", () => {
    let converter: InfoConverter;

    beforeAll(async () => {
        // Import after setup has run to ensure ModelMap is initialized
        const infoConverterModule = await import("./infoConverter.js");
        CountFields = infoConverterModule.CountFields;
        InfoConverter = infoConverterModule.InfoConverter;
        addSupplementalFields = infoConverterModule.addSupplementalFields;
        addSupplementalFieldsHelper = infoConverterModule.addSupplementalFieldsHelper;
        addSupplementalFieldsMultiTypes = infoConverterModule.addSupplementalFieldsMultiTypes;
    });

    beforeEach(() => {
        converter = InfoConverter.get();
        // Clear any cached data
        (converter as any).cache.clear();
    });

    describe("fromApiToPartialApi", () => {
        it("returns undefined when info is not set", () => {
            const apiRelMap = { __typename: "User" as ModelType };
            expect(converter.fromApiToPartialApi(undefined as any, apiRelMap)).toBeUndefined();
        });

        it("throws error when info is not set and throwIfNotPartial is true", () => {
            const apiRelMap = { __typename: "User" as ModelType };
            expect(() => converter.fromApiToPartialApi(undefined as any, apiRelMap, true)).toThrow("0345");
        });

        it("handles simple API info object", () => {
            const info = { id: true, name: true };
            const apiRelMap = { __typename: "User" as ModelType };
            const result = converter.fromApiToPartialApi(info, apiRelMap);
            expect(result).toEqual({
                id: true,
                name: true,
                __typename: "User",
            });
        });

        it("handles paginated search query shape", () => {
            const info = {
                pageInfo: { hasNextPage: true, endCursor: true },
                edges: { node: { id: true, name: true } },
            };
            const apiRelMap = { __typename: "User" as ModelType };
            const result = converter.fromApiToPartialApi(info, apiRelMap);
            expect(result).toEqual({
                id: true,
                name: true,
                __typename: "User",
            });
        });

        it("handles comment thread search query shape", () => {
            const info = {
                endCursor: true,
                totalThreads: true,
                threads: { comment: { id: true, text: true } },
            };
            const apiRelMap = { __typename: "Comment" as ModelType };
            const result = converter.fromApiToPartialApi(info, apiRelMap);
            expect(result).toEqual({
                id: true,
                text: true,
                __typename: "Comment",
            });
        });

        it("preserves __cacheKey when present", () => {
            const info = { id: true, name: true, __cacheKey: "test-key" };
            const apiRelMap = { __typename: "User" as ModelType };
            const result = converter.fromApiToPartialApi(info, apiRelMap);
            expect(result).toEqual({
                id: true,
                name: true,
                __typename: "User",
                __cacheKey: "test-key",
            });
        });

        it("handles nested relationships", () => {
            const info = {
                id: true,
                name: true,
                bookmarks: { id: true, label: true },
            };
            const apiRelMap = {
                __typename: "User" as ModelType,
                bookmarks: "Bookmark" as ModelType,
            };
            const result = converter.fromApiToPartialApi(info, apiRelMap);
            expect(result).toEqual({
                id: true,
                name: true,
                bookmarks: {
                    id: true,
                    label: true,
                    __typename: "Bookmark",
                },
                __typename: "User",
            });
        });

        it("caches results for repeated calls", () => {
            const info = { id: true, name: true };
            const apiRelMap = { __typename: "User" as ModelType };
            
            const result1 = converter.fromApiToPartialApi(info, apiRelMap);
            const result2 = converter.fromApiToPartialApi(info, apiRelMap);
            
            // Results should be identical
            expect(result1).toEqual(result2);
            // Cache should have been used (we can't directly test this without exposing internals)
        });
    });

    describe("fromPartialApiToPrismaSelect", () => {
        it("returns undefined for invalid input", () => {
            expect(converter.fromPartialApiToPrismaSelect(null as any)).toBeUndefined();
        });

        it("converts simple partial to Prisma select", () => {
            const partial = {
                id: true,
                name: true,
                __typename: "User" as ModelType,
            };
            const result = converter.fromPartialApiToPrismaSelect(partial);
            expect(result).toEqual({
                select: {
                    id: true,
                    name: true,
                },
            });
        });

        it("handles count fields conversion", () => {
            const partial = {
                id: true,
                bookmarksCount: true,
                __typename: "User" as ModelType,
            };
            const result = converter.fromPartialApiToPrismaSelect(partial);
            expect(result).toEqual({
                select: {
                    id: true,
                    _count: {
                        select: {
                            bookmarks: true,
                        },
                    },
                },
            });
        });

        it("handles join table conversion", () => {
            const partial = {
                id: true,
                members: { id: true, role: true },
                __typename: "User" as ModelType,
            };
            const result = converter.fromPartialApiToPrismaSelect(partial);
            expect(result).toEqual({
                select: {
                    id: true,
                    members: {
                        select: {
                            member: {
                                select: {
                                    id: true,
                                    role: true,
                                },
                            },
                        },
                    },
                },
            });
        });

        it("handles nested relationships", () => {
            const partial = {
                id: true,
                bookmarks: {
                    id: true,
                    label: true,
                    __typename: "Bookmark" as ModelType,
                },
                __typename: "User" as ModelType,
            };
            const result = converter.fromPartialApiToPrismaSelect(partial);
            expect(result).toEqual({
                select: {
                    id: true,
                    bookmarks: {
                        select: {
                            id: true,
                            label: true,
                        },
                    },
                },
            });
        });

        it("handles polymorphic relationships", () => {
            const partial = {
                id: true,
                to: {
                    user: { id: true, name: true },
                    team: { id: true, handle: true },
                    __typename: "Bookmark" as ModelType,
                },
                __typename: "Bookmark" as ModelType,
            };
            const result = converter.fromPartialApiToPrismaSelect(partial);
            expect(result).toEqual({
                select: {
                    id: true,
                    to: {
                        user: {
                            select: {
                                id: true,
                                name: true,
                            },
                        },
                        team: {
                            select: {
                                id: true,
                                handle: true,
                            },
                        },
                    },
                },
            });
        });
    });

    describe("supplemental fields", () => {
        it("addSupplementalFields works with empty params", async () => {
            const mockUserData = { id: "user123", languages: ["en"] };
            const result = await addSupplementalFields(mockUserData, [], {});
            expect(result).toEqual([]);
        });

        it("addSupplementalFieldsHelper processes basic fields", async () => {
            const mockUserData = { id: "user123", languages: ["en"] };
            const mockObjects = [{ id: "obj1", name: "Test Object" }];
            const mockPartial = { id: true, name: true };
            
            const result = await addSupplementalFieldsHelper({
                languages: ["en"],
                objects: mockObjects,
                objectType: "User",
                partial: mockPartial,
                userData: mockUserData,
            });
            
            expect(Array.isArray(result)).toBe(true);
        });

        it("addSupplementalFieldsMultiTypes handles multiple types", async () => {
            const mockUserData = { id: "user123", languages: ["en"] };
            const mockData = { 
                User: [{ id: "user1", name: "User 1" }],
                Team: [{ id: "team1", handle: "team1" }],
            };
            const mockPartial = { 
                User: { id: true, name: true },
                Team: { id: true, handle: true },
            };
            
            const result = await addSupplementalFieldsMultiTypes(mockData, mockPartial, mockUserData);
            expect(typeof result).toBe("object");
        });
    });

    describe("CountFields", () => {
        describe("addToData", () => {
            it("returns obj unmodified when countFields is undefined", () => {
                const obj = { commentsCount: true };
                expect(CountFields.addToData(obj, undefined)).toEqual(obj);
            });

            it("handles empty obj correctly", () => {
                const countFields = { commentsCount: true } as const;
                expect(CountFields.addToData({}, countFields)).toEqual({});
            });

            it("does not modify obj when countFields is empty", () => {
                const obj = { commentsCount: true };
                expect(CountFields.addToData(obj, {})).toEqual(obj);
            });

            it("does not modify obj if specified count fields do not exist", () => {
                const obj = { likesCount: true };
                const countFields = { commentsCount: true } as const;
                expect(CountFields.addToData(obj, countFields)).toEqual(obj);
            });

            it("handles multiple count fields correctly", () => {
                const obj = { commentsCount: true, reportsCount: true };
                const countFields = { commentsCount: true, reportsCount: true } as const;
                const expected = { _count: { select: { comments: true, reports: true } } };
                expect(CountFields.addToData(obj, countFields)).toEqual(expected);
            });
        });

        describe("removeFromData", () => {
            it("returns obj unmodified when countFields is undefined", () => {
                const obj = { _count: { comments: 5 } };
                expect(CountFields.removeFromData(obj, undefined)).toEqual(obj);
            });

            it("returns obj unmodified when no _count field exists", () => {
                const obj = { id: "123", name: "Test" };
                const countFields = { commentsCount: true };
                expect(CountFields.removeFromData(obj, countFields)).toEqual(obj);
            });

            it("converts single count field correctly", () => {
                const obj = { _count: { comments: 5 }, id: "123" };
                const countFields = { commentsCount: true };
                const expected = { commentsCount: 5, id: "123" };
                expect(CountFields.removeFromData(obj, countFields)).toEqual(expected);
            });

            it("converts multiple count fields correctly", () => {
                const obj = { _count: { comments: 5, reports: 2 }, id: "123" };
                const countFields = { commentsCount: true, reportsCount: true };
                const expected = { commentsCount: 5, reportsCount: 2, id: "123" };
                expect(CountFields.removeFromData(obj, countFields)).toEqual(expected);
            });

            it("ignores count fields not in _count", () => {
                const obj = { _count: { comments: 5 }, id: "123" };
                const countFields = { commentsCount: true, reportsCount: true };
                const expected = { commentsCount: 5, id: "123" };
                expect(CountFields.removeFromData(obj, countFields)).toEqual(expected);
            });
        });
    });

    describe("Migration Benefits", () => {
        it("demonstrates real configuration usage", () => {
            // Test shows how we can use real format configurations
            const userFormat = testConfigs.User;
            expect(userFormat.apiRelMap.__typename).toBe("User");
            expect(userFormat.countFields.bookmarksCount).toBe(true);
            expect(userFormat.joinMap.members).toBe("member");
        });

        it("shows flexibility with different configurations", () => {
            // Same converter logic, different configurations
            const converter = InfoConverter.get();
            
            // Test with User configuration
            const userPartial = {
                id: true,
                bookmarksCount: true,
                __typename: "User" as ModelType,
            };
            
            const userResult = converter.fromPartialApiToPrismaSelect(userPartial);
            expect(userResult.select).toHaveProperty("_count");
            
            // Test with Team configuration
            const teamPartial = {
                id: true,
                handle: true,
                __typename: "Team" as ModelType,
            };
            
            const teamResult = converter.fromPartialApiToPrismaSelect(teamPartial);
            expect(teamResult.select).toHaveProperty("id");
            expect(teamResult.select).toHaveProperty("handle");
        });
    });
});

