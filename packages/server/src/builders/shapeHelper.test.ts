/**
 * shapeHelper tests - migrated from mocking to real dependencies
 * 
 * Tests the shapeHelper function which transforms API mutation data into 
 * Prisma relationship operations. Uses real ID generation for authenticity.
 */

import { describe, expect, it, beforeEach, beforeAll } from "vitest";
import { type ModelType, generatePK, type SessionUser, SEEDED_PUBLIC_IDS, AccountStatus } from "@vrooli/shared";
import { DbProvider } from "../db/provider.js";

// Delay imports that use ModelMap to avoid initialization issues
/* eslint-disable @typescript-eslint/consistent-type-imports */
let shapeHelper: typeof import("./shapeHelper.js").shapeHelper;
let addRels: typeof import("./shapeHelper.js").addRels;
let updateRels: typeof import("./shapeHelper.js").updateRels;
type ShapeHelperProps<T, U> = import("./shapeHelper.js").ShapeHelperProps<T, U>;
/* eslint-enable @typescript-eslint/consistent-type-imports */


describe("shapeHelper", () => {
    let defaultProps: ShapeHelperProps<false, typeof addRels>;
    
    beforeAll(async () => {
        // Create admin user that some shapeHelper operations require
        await DbProvider.get().user.create({
            data: {
                id: BigInt("111111111111111111"),
                publicId: SEEDED_PUBLIC_IDS.Admin,
                name: "Admin User",
                handle: "admin",
                status: AccountStatus.Unlocked,
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
        });
        
        // Import after setup has run to ensure ModelMap is initialized
        const shapeHelperModule = await import("./shapeHelper.js");
        shapeHelper = shapeHelperModule.shapeHelper;
        addRels = shapeHelperModule.addRels;
        updateRels = shapeHelperModule.updateRels;
        
        // Initialize default props after imports
        const mockUserData: SessionUser = {
            id: "123",
            languages: ["en"],
        };
        
        defaultProps = {
            additionalData: {},
            data: {},
            isOneToOne: false,
            objectType: "Bookmark",
            parentRelationshipName: "parent",
            preMap: {
                Bookmark: {
                    embeddingNeedsUpdateMap: {},
                },
            },
            relation: "bookmarks",
            relTypes: addRels,
            userData: mockUserData,
        };
    });

    // Generate test IDs for predictable testing - let TypeScript infer string type
    let testId1;
    let testId2;
    let testId3;
    
    beforeEach(() => {
        // Generate fresh IDs for each test
        testId1 = generatePK().toString();
        testId2 = generatePK().toString();
        testId3 = generatePK().toString();
    });

    describe("basic operations", () => {
        it("returns undefined for empty data", async () => {
            const result = await shapeHelper(defaultProps);
            expect(result).toBeUndefined();
        });

        it("handles connect operation", async () => {
            const props: ShapeHelperProps<false, ["Connect"]> = {
                ...defaultProps,
                data: {
                    bookmarksConnect: ["123", "456"],
                },
                relTypes: ["Connect"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                connect: [
                    { id: BigInt("123") },
                    { id: BigInt("456") },
                ],
            });
        });

        it("handles create operation", async () => {
            const props: ShapeHelperProps<false, ["Create"]> = {
                ...defaultProps,
                data: {
                    bookmarksCreate: [
                        { id: "1", bookmarkFor: "Resource", forConnect: "101" },
                        { id: "2", bookmarkFor: "Resource", forConnect: "102" },
                    ],
                },
                preMap: {
                    Bookmark: {
                        embeddingNeedsUpdateMap: {
                            "1": false,
                            "2": false,
                        },
                    },
                },
                relTypes: ["Create"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                create: expect.arrayContaining([
                    expect.objectContaining({
                        id: BigInt("1"),
                        resource: { connect: { id: BigInt("101") } },
                    }),
                    expect.objectContaining({
                        id: BigInt("2"),
                        resource: { connect: { id: BigInt("102") } },
                    }),
                ]),
            });
        });

        it("handles disconnect operation", async () => {
            const props: ShapeHelperProps<false, ["Disconnect"]> = {
                ...defaultProps,
                data: {
                    bookmarksDisconnect: ["123", "456"],
                },
                relTypes: ["Disconnect"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                disconnect: [
                    { id: BigInt("123") },
                    { id: BigInt("456") },
                ],
            });
        });

        it("handles delete operation with soft delete", async () => {
            const props: ShapeHelperProps<false, ["Delete"], true> = {
                ...defaultProps,
                data: {
                    bookmarksDelete: ["123", "456"],
                },
                relTypes: ["Delete"],
                softDelete: true,
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                update: [
                    { where: { id: BigInt("123") }, data: { isDeleted: true } },
                    { where: { id: BigInt("456") }, data: { isDeleted: true } },
                ],
            });
        });

        it("handles delete operation without soft delete", async () => {
            const props: ShapeHelperProps<false, ["Delete"]> = {
                ...defaultProps,
                data: {
                    bookmarksDelete: ["123", "456"],
                },
                relTypes: ["Delete"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                delete: [
                    { id: BigInt("123") },
                    { id: BigInt("456") },
                ],
            });
        });

        it("handles update operation", async () => {
            const props: ShapeHelperProps<false, ["Update"]> = {
                ...defaultProps,
                data: {
                    bookmarksUpdate: [
                        { id: "123", label: "Updated 1" },
                        { id: "456", label: "Updated 2" },
                    ],
                },
                relTypes: ["Update"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                update: [
                    { where: { id: BigInt("123") }, data: { label: "Updated 1" } },
                    { where: { id: BigInt("456") }, data: { label: "Updated 2" } },
                ],
            });
        });
    });

    describe("one-to-one relationships", () => {
        it("handles connect for one-to-one", async () => {
            const props: ShapeHelperProps<true, ["Connect"]> = {
                ...defaultProps,
                data: {
                    profileConnect: "123",
                },
                relation: "profile",
                relTypes: ["Connect"],
                isOneToOne: true,
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                connect: { id: BigInt("123") },
            });
        });

        it("handles create for one-to-one", async () => {
            const props: ShapeHelperProps<true, ["Create"]> = {
                ...defaultProps,
                data: {
                    profileCreate: { id: "1", bio: "My bio" },
                },
                preMap: {
                    Bookmark: {
                        embeddingNeedsUpdateMap: {
                            "1": false,
                        },
                    },
                },
                relation: "profile",
                relTypes: ["Create"],
                isOneToOne: true,
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                create: { id: "1", bio: "My bio" },
            });
        });

        it("handles disconnect for one-to-one", async () => {
            const props: ShapeHelperProps<true, ["Disconnect"]> = {
                ...defaultProps,
                data: {
                    profileDisconnect: true,
                },
                relation: "profile",
                relTypes: ["Disconnect"],
                isOneToOne: true,
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                disconnect: true,
            });
        });
    });

    describe("join table operations", () => {
        it("handles join table create", async () => {
            const props: ShapeHelperProps<false, ["Create"]> = {
                ...defaultProps,
                data: {
                    membersCreate: [
                        { userId: "111111111111111111", role: "Admin" },
                        { userId: "222222222222222222", role: "Member" },
                    ],
                },
                joinData: {
                    fieldName: "member",
                    uniqueFieldName: "userId",
                },
                relation: "members",
                relTypes: ["Create"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                create: expect.arrayContaining([
                    expect.objectContaining({
                        member: { 
                            create: expect.objectContaining({ 
                                id: expect.any(String),
                                role: "Admin", 
                            }), 
                        },
                        user: { connect: { id: BigInt("111111111111111111") } },
                    }),
                    expect.objectContaining({
                        member: { 
                            create: expect.objectContaining({ 
                                id: expect.any(String),
                                role: "Member", 
                            }), 
                        },
                        user: { connect: { id: BigInt("222222222222222222") } },
                    }),
                ]),
            });
        });

        it("handles join table update", async () => {
            const props: ShapeHelperProps<false, ["Update"]> = {
                ...defaultProps,
                data: {
                    membersUpdate: [
                        { id: "123", role: "Updated Admin" },
                        { id: "456", role: "Updated Member" },
                    ],
                },
                joinData: {
                    fieldName: "member",
                    uniqueFieldName: "userId",
                },
                relation: "members",
                relTypes: ["Update"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                update: [
                    {
                        where: { id: BigInt("123") },
                        data: {
                            member: {
                                update: { role: "Updated Admin" },
                            },
                        },
                    },
                    {
                        where: { id: BigInt("456") },
                        data: {
                            member: {
                                update: { role: "Updated Member" },
                            },
                        },
                    },
                ],
            });
        });
    });

    describe("complex operations", () => {
        it("handles multiple operation types", async () => {
            const props: ShapeHelperProps<false, ["Connect", "Create", "Update", "Disconnect"]> = {
                ...defaultProps,
                data: {
                    bookmarksConnect: ["12345"],
                    bookmarksCreate: [{ id: "1", label: "New Bookmark" }],
                    bookmarksUpdate: [{ id: "67890", label: "Updated Bookmark" }],
                    bookmarksDisconnect: ["11111"],
                },
                preMap: {
                    Bookmark: {
                        embeddingNeedsUpdateMap: {
                            "1": false,
                            "67890": false,
                        },
                    },
                },
                relTypes: ["Connect", "Create", "Update", "Disconnect"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                connect: [{ id: BigInt("12345") }],
                create: [{ id: "1", label: "New Bookmark" }],
                update: [
                    {
                        where: { id: BigInt("67890") },
                        data: { label: "Updated Bookmark" },
                    },
                ],
                disconnect: [{ id: BigInt("11111") }],
            });
        });

        it("handles idsCreateToConnect conversion", async () => {
            const connectId = generatePK().toString();
            const createId = generatePK().toString();
            
            const props: ShapeHelperProps<false, ["Create"]> = {
                ...defaultProps,
                data: {
                    bookmarksCreate: [
                        { id: connectId, label: "Will be connected" },
                        { id: createId, label: "Will be created" },
                    ],
                },
                preMap: {
                    Bookmark: {
                        embeddingNeedsUpdateMap: {
                            [connectId]: false,
                            [createId]: false,
                        },
                    },
                },
                idsCreateToConnect: {
                    [connectId]: connectId,
                },
                relTypes: ["Create"],
            };

            const result = await shapeHelper(props);

            expect(result).toEqual({
                connect: [{ id: BigInt(connectId) }],
                create: [{ id: createId, label: "Will be created" }],
            });
        });

        it("validates IDs when specified", async () => {
            const props: ShapeHelperProps<false, ["Connect"]> = {
                ...defaultProps,
                data: {
                    bookmarksConnect: ["12345", "", "67890"],
                },
                relTypes: ["Connect"],
            };

            const result = await shapeHelper(props);

            // Empty string should be filtered out by validatePK
            expect(result).toEqual({
                connect: [
                    { id: BigInt("12345") },
                    { id: BigInt("67890") },
                ],
            });
        });
    });

    describe("error handling", () => {
        it("throws error for invalid operation type", async () => {
            const props = {
                ...defaultProps,
                data: {
                    bookmarksInvalid: ["123"],
                },
                relTypes: ["Invalid" as any],
            };

            await expect(shapeHelper(props)).rejects.toThrow();
        });

        it("throws error when data type doesn't match operation", async () => {
            const props: ShapeHelperProps<false, ["Connect"]> = {
                ...defaultProps,
                data: {
                    bookmarksConnect: "not-an-array", // Should be array for many relationship
                },
                relTypes: ["Connect"],
            };

            await expect(shapeHelper(props)).rejects.toThrow();
        });
    });

    describe("Migration Benefits", () => {

        it("shows real shape operations without ModelMap mocking", async () => {
            // The shapeHelper function works with real logic
            // No need to mock ModelMap - the function handles the data transformation
            const props: ShapeHelperProps<false, ["Create"]> = {
                ...defaultProps,
                data: {
                    tagsCreate: [{ id: "1", name: "JavaScript" }],
                },
                preMap: {
                    Bookmark: {
                        embeddingNeedsUpdateMap: {
                            "1": false,
                        },
                    },
                },
                relation: "tags",
                relTypes: ["Create"],
            };

            const result = await shapeHelper(props);
            expect(result).toEqual({
                create: [{ id: "1", name: "JavaScript" }],
            });
        });
    });
});
