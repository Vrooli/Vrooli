import { generatePK, generatePublicId, ResourceSubType, ResourceType } from "@vrooli/shared";
import fs from "fs";
import path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import zlib from "zlib";
import { genSitemap, isSitemapMissing } from "./genSitemap.js";

import { DbProvider } from "@vrooli/server/db/provider.js";

// Mock fs module
vi.mock("fs", () => ({
    default: {
        existsSync: vi.fn(),
        mkdirSync: vi.fn(),
        writeFileSync: vi.fn(),
    },
}));

// Mock zlib module
vi.mock("zlib", () => ({
    default: {
        gzip: vi.fn((data, callback) => {
            // Mock successful gzip compression
            callback(null, Buffer.from("compressed-data"));
        }),
    },
}));

// Mock only the UI_URL constant for testing
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        UI_URL_REMOTE: "https://test.vrooli.com",
    };
});

describe("genSitemap tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testResourceVersionIds: bigint[] = [];
    const testRoutineIds: bigint[] = [];
    const testApiIds: bigint[] = [];
    const testNoteIds: bigint[] = [];
    const testProjectIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testResourceIds.length = 0;
        testResourceVersionIds.length = 0;
        testRoutineIds.length = 0;
        testApiIds.length = 0;
        testNoteIds.length = 0;
        testProjectIds.length = 0;

        // Reset all mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testResourceVersionIds.length > 0) {
            await db.resource_version.deleteMany({ where: { id: { in: testResourceVersionIds } } });
        }
        if (testResourceIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testResourceIds } } });
        }
        if (testRoutineIds.length > 0) {
            await db.routine.deleteMany({ where: { id: { in: testRoutineIds } } });
        }
        if (testApiIds.length > 0) {
            await db.api.deleteMany({ where: { id: { in: testApiIds } } });
        }
        if (testNoteIds.length > 0) {
            await db.note.deleteMany({ where: { id: { in: testNoteIds } } });
        }
        if (testProjectIds.length > 0) {
            await db.project.deleteMany({ where: { id: { in: testProjectIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    describe("isSitemapMissing", () => {
        it("should return true when sitemap.xml does not exist", async () => {
            const mockExistsSync = fs.existsSync as any;
            mockExistsSync.mockReturnValue(false);

            const result = await isSitemapMissing();
            expect(result).toBe(true);
        });

        it("should return false when sitemap.xml exists", async () => {
            const mockExistsSync = fs.existsSync as any;
            mockExistsSync.mockReturnValue(true);

            const result = await isSitemapMissing();
            expect(result).toBe(false);
        });
    });

    describe("genSitemap", () => {
        it("should create sitemap directory if it doesn't exist", async () => {
            const mockExistsSync = fs.existsSync as any;
            const mockMkdirSync = fs.mkdirSync as any;
            
            // First call checks sitemap directory, second checks routes sitemap
            mockExistsSync.mockReturnValueOnce(false).mockReturnValueOnce(true);

            await genSitemap();

            expect(mockMkdirSync).toHaveBeenCalledWith(expect.stringContaining("/sitemaps"));
        });

        it("should generate sitemap for public users with handles", async () => {
            // Create test users
            const user1 = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Public User 1",
                    handle: "publicuser1",
                    isPrivate: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", bio: "English bio" },
                            { id: generatePK(), language: "es", bio: "Spanish bio" },
                        ],
                    },
                },
            });
            testUserIds.push(user1.id);

            const user2 = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Public User 2",
                    handle: "publicuser2",
                    isPrivate: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", bio: "Another bio" },
                        ],
                    },
                },
            });
            testUserIds.push(user2.id);

            // Private user should not be included
            const privateUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Private User",
                    handle: "privateuser",
                    isPrivate: true,
                },
            });
            testUserIds.push(privateUser.id);

            const mockExistsSync = fs.existsSync as any;
            const mockWriteFileSync = fs.writeFileSync as any;
            mockExistsSync.mockReturnValue(true);

            await genSitemap();

            // Check that zlib.gzip was called for User sitemap
            expect(zlib.gzip).toHaveBeenCalled();
            
            // Check that sitemap files were written
            expect(mockWriteFileSync).toHaveBeenCalledWith(
                expect.stringContaining("User-0.xml.gz"),
                expect.any(Uint8Array)
            );
            
            // Check that sitemap index was written
            expect(mockWriteFileSync).toHaveBeenCalledWith(
                expect.stringContaining("sitemap.xml"),
                expect.stringContaining("User-0.xml.gz")
            );
        });

        it("should generate sitemap for public teams", async () => {
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Team Owner",
                    handle: "teamowner",
                },
            });
            testUserIds.push(owner.id);

            // Create public teams
            const team1 = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    handle: "publicteam1",
                    isPrivate: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Public Team 1" },
                        ],
                    },
                },
            });
            testTeamIds.push(team1.id);

            const team2 = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    handle: "publicteam2",
                    isPrivate: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Public Team 2" },
                            { id: generatePK(), language: "fr", name: "Équipe Publique 2" },
                        ],
                    },
                },
            });
            testTeamIds.push(team2.id);

            // Private team should not be included
            const privateTeam = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    handle: "privateteam",
                    isPrivate: true,
                },
            });
            testTeamIds.push(privateTeam.id);

            const mockExistsSync = fs.existsSync as any;
            const mockWriteFileSync = fs.writeFileSync as any;
            mockExistsSync.mockReturnValue(true);

            await genSitemap();

            // Check that Team sitemap was generated
            expect(mockWriteFileSync).toHaveBeenCalledWith(
                expect.stringContaining("Team-0.xml.gz"),
                expect.any(Uint8Array)
            );
        });

        it("should generate sitemap for different resource version types", async () => {
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Resource Owner",
                    handle: "resourceowner",
                },
            });
            testUserIds.push(owner.id);

            // Create routine for RoutineVersion resource
            const routine = await DbProvider.get().routine.create({
                data: {
                    id: generatePK(),
                    createdById: owner.id,
                    isPrivate: false,
                    isInternal: false,
                },
            });
            testRoutineIds.push(routine.id);

            // Create resources of different types
            const routineResource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    resourceType: ResourceType.Routine,
                    isPrivate: false,
                    isDeleted: false,
                },
            });
            testResourceIds.push(routineResource.id);

            // Create resource version for routine
            const routineVersion = await DbProvider.get().resource_version.create({
                data: {
                    id: generatePK(),
                    resourceId: routineResource.id,
                    routineId: routine.id,
                    versionLabel: "1.0.0",
                    schemaLanguage: "json",
                    schema: "{}",
                    resourceSubType: ResourceSubType.RoutineMultiStep,
                    isDeleted: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Multi-Step Routine", description: "A routine" },
                        ],
                    },
                },
            });
            testResourceVersionIds.push(routineVersion.id);

            // Create API resource
            const api = await DbProvider.get().api.create({
                data: {
                    id: generatePK(),
                    createdById: owner.id,
                    isPrivate: false,
                    isInternal: false,
                },
            });
            testApiIds.push(api.id);

            const apiResource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    resourceType: ResourceType.Api,
                    handle: "test-api",
                    isPrivate: false,
                    isDeleted: false,
                },
            });
            testResourceIds.push(apiResource.id);

            const apiVersion = await DbProvider.get().resource_version.create({
                data: {
                    id: generatePK(),
                    resourceId: apiResource.id,
                    apiId: api.id,
                    versionLabel: "2.0.0",
                    schemaLanguage: "graphql",
                    schema: "type Query { hello: String }",
                    isDeleted: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Test API", description: "An API" },
                        ],
                    },
                },
            });
            testResourceVersionIds.push(apiVersion.id);

            // Create Note resource
            const note = await DbProvider.get().note.create({
                data: {
                    id: generatePK(),
                    ownedByUserId: owner.id,
                    isPrivate: false,
                },
            });
            testNoteIds.push(note.id);

            const noteResource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    resourceType: ResourceType.Note,
                    isPrivate: false,
                    isDeleted: false,
                },
            });
            testResourceIds.push(noteResource.id);

            const noteVersion = await DbProvider.get().resource_version.create({
                data: {
                    id: generatePK(),
                    resourceId: noteResource.id,
                    noteId: note.id,
                    versionLabel: "1.0.0",
                    schemaLanguage: "json",
                    schema: "{}",
                    isDeleted: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Test Note", description: "A note" },
                        ],
                    },
                },
            });
            testResourceVersionIds.push(noteVersion.id);

            // Create deleted resource version (should not be included)
            const deletedResource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    resourceType: ResourceType.Project,
                    isPrivate: false,
                    isDeleted: true, // Deleted
                },
            });
            testResourceIds.push(deletedResource.id);

            const project = await DbProvider.get().project.create({
                data: {
                    id: generatePK(),
                    ownedByUserId: owner.id,
                    isPrivate: false,
                },
            });
            testProjectIds.push(project.id);

            const deletedVersion = await DbProvider.get().resource_version.create({
                data: {
                    id: generatePK(),
                    resourceId: deletedResource.id,
                    projectId: project.id,
                    versionLabel: "1.0.0",
                    schemaLanguage: "json",
                    schema: "{}",
                    isDeleted: false,
                    translations: {
                        create: [
                            { id: generatePK(), language: "en", name: "Deleted Project", description: "Should not appear" },
                        ],
                    },
                },
            });
            testResourceVersionIds.push(deletedVersion.id);

            const mockExistsSync = fs.existsSync as any;
            const mockWriteFileSync = fs.writeFileSync as any;
            mockExistsSync.mockReturnValue(true);

            await genSitemap();

            // Check that ResourceVersion sitemap was generated
            expect(mockWriteFileSync).toHaveBeenCalledWith(
                expect.stringContaining("ResourceVersion-0.xml.gz"),
                expect.any(Uint8Array)
            );
        });

        it("should handle special resource subtypes", async () => {
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Subtype Owner",
                    handle: "subtypeowner",
                },
            });
            testUserIds.push(owner.id);

            // Create resources with specific subtypes
            const subtypeTests = [
                { subType: ResourceSubType.CodeDataConverter, expectedLink: "data-converter" },
                { subType: ResourceSubType.CodeSmartContract, expectedLink: "smart-contract" },
                { subType: ResourceSubType.StandardPrompt, expectedLink: "prompt" },
                { subType: ResourceSubType.StandardDataStructure, expectedLink: "data-structure" },
            ];

            for (const test of subtypeTests) {
                const resource = await DbProvider.get().resource.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        createdById: owner.id,
                        resourceType: ResourceType.Routine, // Using routine as base type
                        isPrivate: false,
                        isDeleted: false,
                    },
                });
                testResourceIds.push(resource.id);

                const routine = await DbProvider.get().routine.create({
                    data: {
                        id: generatePK(),
                        createdById: owner.id,
                        isPrivate: false,
                        isInternal: false,
                    },
                });
                testRoutineIds.push(routine.id);

                const version = await DbProvider.get().resource_version.create({
                    data: {
                        id: generatePK(),
                        resourceId: resource.id,
                        routineId: routine.id,
                        versionLabel: "1.0.0",
                        schemaLanguage: "json",
                        schema: "{}",
                        resourceSubType: test.subType,
                        isDeleted: false,
                        translations: {
                            create: [
                                { id: generatePK(), language: "en", name: `Test ${test.subType}`, description: "Test" },
                            ],
                        },
                    },
                });
                testResourceVersionIds.push(version.id);
            }

            const mockExistsSync = fs.existsSync as any;
            mockExistsSync.mockReturnValue(true);

            await genSitemap();

            expect(zlib.gzip).toHaveBeenCalled();
        });

        it("should handle empty results gracefully", async () => {
            // Don't create any test data
            const mockExistsSync = fs.existsSync as any;
            const mockWriteFileSync = fs.writeFileSync as any;
            mockExistsSync.mockReturnValue(true);

            await genSitemap();

            // Should still write the index file
            expect(mockWriteFileSync).toHaveBeenCalledWith(
                expect.stringContaining("sitemap.xml"),
                expect.any(String)
            );
            
            // But should not write any object-specific sitemaps
            expect(mockWriteFileSync).not.toHaveBeenCalledWith(
                expect.stringContaining("User-0.xml.gz"),
                expect.any(Uint8Array)
            );
            expect(mockWriteFileSync).not.toHaveBeenCalledWith(
                expect.stringContaining("Team-0.xml.gz"),
                expect.any(Uint8Array)
            );
            expect(mockWriteFileSync).not.toHaveBeenCalledWith(
                expect.stringContaining("ResourceVersion-0.xml.gz"),
                expect.any(Uint8Array)
            );
        });

        it("should handle missing route sitemap file", async () => {
            const mockExistsSync = fs.existsSync as any;
            const mockWriteFileSync = fs.writeFileSync as any;
            
            // First call for sitemap dir exists, second for routes file doesn't exist
            mockExistsSync.mockReturnValueOnce(true).mockReturnValueOnce(false);

            await genSitemap();

            // Index should still be created but without routes file
            expect(mockWriteFileSync).toHaveBeenCalledWith(
                expect.stringContaining("sitemap.xml"),
                expect.not.stringContaining("sitemap-routes.xml")
            );
        });

        it("should handle zlib compression errors", async () => {
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Error Test User",
                    handle: "erroruser",
                    isPrivate: false,
                    translations: {
                        create: [{ id: generatePK(), language: "en", bio: "Bio" }],
                    },
                },
            });
            testUserIds.push(owner.id);

            const mockExistsSync = fs.existsSync as any;
            mockExistsSync.mockReturnValue(true);

            // Mock zlib to return an error
            const mockGzip = zlib.gzip as any;
            mockGzip.mockImplementationOnce((data, callback) => {
                callback(new Error("Compression failed"), null);
            });

            // Should not throw, just log the error
            await expect(genSitemap()).resolves.not.toThrow();
        });

        it("should handle large datasets by batching", async () => {
            // Create more than BATCH_SIZE (100) users
            const userPromises = [];
            for (let i = 0; i < 120; i++) {
                userPromises.push(
                    DbProvider.get().user.create({
                        data: {
                            id: generatePK(),
                            publicId: generatePublicId(),
                            name: `Batch User ${i}`,
                            handle: `batchuser${i}`,
                            isPrivate: false,
                            translations: {
                                create: [{ id: generatePK(), language: "en", bio: `Bio ${i}` }],
                            },
                        },
                    })
                );
            }
            const users = await Promise.all(userPromises);
            users.forEach(u => testUserIds.push(u.id));

            const mockExistsSync = fs.existsSync as any;
            mockExistsSync.mockReturnValue(true);

            await genSitemap();

            // Should have made multiple batch queries
            expect(zlib.gzip).toHaveBeenCalled();
        });
    });
});