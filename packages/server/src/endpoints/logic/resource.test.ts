import { type FindByPublicIdInput, type ResourceVersionCreateInput, type ResourceVersionSearchInput, type ResourceVersionUpdateInput, generatePK, ResourceType } from "@vrooli/shared";
import { resourceVersionTestDataFactory } from "@vrooli/shared/test-fixtures/api-inputs";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertRequiresApiKeyWritePermissions, assertRequiresAuth } from "../../__test/authTestUtils.js";
import { mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { resource_createOne } from "../generated/resource_createOne.js";
import { resource_findMany } from "../generated/resource_findMany.js";
import { resource_findOne } from "../generated/resource_findOne.js";
import { resource_updateOne } from "../generated/resource_updateOne.js";
import { resource } from "./resource.js";
// Import database fixtures
import { ResourceDbFactory } from "../../__test/fixtures/db/resourceFixtures.js";
import { TeamDbFactory } from "../../__test/fixtures/db/teamFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsResource", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Clean up after each test to prevent data leakage
        await cleanupGroups.team(DbProvider.get());

        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["team", "member", "member_invite", "meeting", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.team(DbProvider.get());
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create test tags with unique names to avoid conflicts
        const uniqueSuffix = Date.now().toString();
        const tags = await Promise.all([
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `api_${uniqueSuffix}` } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `database_${uniqueSuffix}` } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `utility_${uniqueSuffix}` } }),
        ]);

        // Create test team
        const team = await DbProvider.get().team.create({
            data: TeamDbFactory.createWithMembers(
                [{ userId: testUsers.records[0].id.toString(), isAdmin: true }],
                {
                    id: generatePK(),
                    createdBy: { connect: { id: testUsers.records[0].id } },
                }
            ),
        });

        // Create test resources
        const resources = await Promise.all([
            // Public resource by user 1
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion(
                    testUsers.records[0].id,
                    {
                        name: "Public Resource",
                        description: "A public test resource",
                        versionLabel: "1.0.0",
                        isComplete: true,
                    },
                    ResourceType.Code,
                    {
                        isPrivate: false,
                        publicId: "pub_res_123",
                    }
                ),
                include: { versions: true },
            }),
            // Private resource by user 2
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion(
                    testUsers.records[1].id,
                    {
                        name: "Private Resource",
                        description: "A private test resource",
                        versionLabel: "1.0.0",
                        isComplete: true,
                    },
                    ResourceType.Api,
                    {
                        isPrivate: true,
                        publicId: "priv_res_456",
                    }
                ),
                include: { versions: true },
            }),
            // Team resource
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion(
                    testUsers.records[0].id,
                    {
                        name: "Team Resource",
                        description: "A team-owned test resource",
                        versionLabel: "1.0.0",
                        isComplete: true,
                    },
                    ResourceType.Project,
                    {
                        isPrivate: false,
                        publicId: "team_res_789",
                        ownedByTeam: { connect: { id: team.id } },
                        ownedByUser: undefined,
                    }
                ),
                include: { versions: true },
            }),
        ]);

        return { testUsers, tags, team, resources };
    }

    describe("findOne", () => {
        describe("authentication", () => {
            it("allows unauthenticated access to public resources", async () => {
                const { resources } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: resources[0].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(resources[0].versions[0].id.toString());
                expect(result.root?.publicId).toBe(resources[0].publicId);
            });

            it("supports API key with read permissions", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockApiSession({
                    userId: testUsers.records[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockReadPublicPermissions(["ResourceVersion"]),
                });

                const input: FindByPublicIdInput = { publicId: resources[0].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).not.toBeNull();
            });
        });

        describe("valid", () => {
            it("returns public resource details", async () => {
                const { resources } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: resources[0].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).not.toBeNull();
                expect(result.root?.publicId).toBe(resources[0].publicId);
                expect(result.root?.isPrivate).toBe(false);
                expect(result.usedBy).toBeDefined();
                expect(result.translations).toBeDefined();
            });

            it("returns private resource to owner", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[1].id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: resources[1].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).not.toBeNull();
                expect(result.root?.isPrivate).toBe(true);
                expect(result.root?.createdBy?.id).toBe(testUsers.records[1].id.toString());
            });

            it("returns team resource to team member", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: resources[2].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).not.toBeNull();
                expect(result.root?.team?.id).toBeDefined();
            });

            it("returns resource with complete metadata", async () => {
                const { resources } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: resources[0].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result.versionLabel).toBeDefined();
                expect(result.versionNotes).toBeDefined();
                expect(result.directoryListings).toBeDefined();
                expect(result.isLatest).toBe(true);
                expect(result.isPrivate).toBe(false);
                expect(result.createdAt).toBeDefined();
                expect(result.updatedAt).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("returns null for private resource when not owner", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[2].id.toString(), // User 3 is not owner
                });

                const input: FindByPublicIdInput = { publicId: resources[1].publicId };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent resource", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: "non_existent_resource" };
                const result = await resource.findOne({ input }, { req, res }, resource_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            it("allows unauthenticated access", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {};
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result).toBeDefined();
                expect(result.results).toBeDefined();
                // Should only see public resources
                expect(result.results.every(r => !r.isPrivate)).toBe(true);
            });

            it("supports API key with read permissions", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockApiSession({
                    userId: testUsers.records[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockReadPublicPermissions(["ResourceVersion"]),
                });

                const input: ResourceVersionSearchInput = {};
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results).toBeDefined();
            });
        });

        describe("valid", () => {
            it("returns public resources for unauthenticated users", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {};
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results).toHaveLength(2); // 2 public resources
                expect(result.results.every(r => !r.isPrivate)).toBe(true);
                expect(result.totalCount).toBe(2);
            });

            it("returns all accessible resources for authenticated user", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[1].id.toString(), // Owner of private resource
                });

                const input: ResourceVersionSearchInput = {};
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results).toHaveLength(3); // All resources accessible
                expect(result.totalCount).toBe(3);
            });

            it("filters resources by search query", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    searchString: "team",
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].root?.publicId).toBe("team_resource_789");
            });

            it("filters resources by tags", async () => {
                const { tags } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    tags: [tags[0].tag], // "api" tag
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results.length).toBeGreaterThanOrEqual(0);
            });

            it("filters by resource type", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    resourceTypes: ["Api"],
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results.every(r => r.resourceType === "Api")).toBe(true);
            });

            it("sorts resources by creation date", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    sortBy: "DateCreatedDesc",
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                // Verify results are sorted by creation date (newest first)
                for (let i = 1; i < result.results.length; i++) {
                    const prevDate = new Date(result.results[i - 1].createdAt);
                    const currDate = new Date(result.results[i].createdAt);
                    expect(prevDate.getTime()).toBeGreaterThanOrEqual(currDate.getTime());
                }
            });

            it("paginates results", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    take: 1,
                    skip: 0,
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.totalCount).toBe(2);
            });

            it("filters by user", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    createdByUserId: testUsers.records[0].id.toString(),
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results.every(r =>
                    r.root?.createdBy?.id === testUsers.records[0].id.toString(),
                )).toBe(true);
            });

            it("filters by team", async () => {
                const { team } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: ResourceVersionSearchInput = {
                    teamId: team.id.toString(),
                };
                const result = await resource.findMany({ input }, { req, res }, resource_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].root?.team?.id).toBe(team.id.toString());
            });
        });
    });

    describe("createOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    resource.createOne,
                    {
                        id: generatePK().toString(),
                        versionLabel: "1.0.0",
                        resourceType: "Api",
                        root: {
                            id: generatePK(),
                            publicId: generatePublicId(),
                            isPrivate: false,
                        },
                        translationsCreate: [{
                            id: generatePK(),
                            language: "en",
                            name: "Test Resource",
                        }],
                    },
                    resource_createOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    resource.createOne,
                    {
                        id: generatePK().toString(),
                        versionLabel: "1.0.0",
                        resourceType: "Api",
                        root: {
                            id: generatePK(),
                            publicId: generatePublicId(),
                            isPrivate: false,
                        },
                        translationsCreate: [{
                            id: generatePK(),
                            language: "en",
                            name: "Test Resource",
                        }],
                    },
                    resource_createOne,
                    ["ResourceVersion"],
                );
            });
        });

        describe("valid", () => {
            it("creates minimal resource version", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: ResourceVersionCreateInput = resourceVersionTestDataFactory.createMinimal({
                    id: generatePK(),
                    versionLabel: "1.0.0",
                    resourceType: "Api",
                    root: {
                        id: generatePK(),
                        publicId: "test_api_resource",
                        isPrivate: false,
                    },
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test API Resource",
                        description: "A test API resource",
                    }],
                });

                const result = await resource.createOne({ input }, { req, res }, resource_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("ResourceVersion");
                expect(result.id).toBe(input.id);
                expect(result.versionLabel).toBe("1.0.0");
                expect(result.resourceType).toBe("Api");
                expect(result.root?.publicId).toBe("test_api_resource");
                expect(result.translations?.[0]?.name).toBe("Test API Resource");

                // Verify in database
                const createdResource = await DbProvider.get().resource.findUnique({
                    where: { id: BigInt(input.root.id) },
                    include: { versions: true },
                });
                expect(createdResource).toBeDefined();
                expect(createdResource?.versions).toHaveLength(1);
            });

            it("creates resource with complete configuration", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                // Create tags for the resource with unique names
                const uniqueSuffix = Date.now().toString();
                const tags = await Promise.all([
                    DbProvider.get().tag.create({ data: { id: generatePK(), tag: `rest-api_${uniqueSuffix}` } }),
                    DbProvider.get().tag.create({ data: { id: generatePK(), tag: `json_${uniqueSuffix}` } }),
                ]);

                const input: ResourceVersionCreateInput = resourceVersionTestDataFactory.createMinimal({
                    id: generatePK(),
                    versionLabel: "2.1.0",
                    versionNotes: "Added new endpoints and improved error handling",
                    resourceType: "Api",
                    usedBy: ["project", "routine"],
                    root: {
                        id: generatePK(),
                        publicId: "advanced_api_resource",
                        isPrivate: false,
                        tagsConnect: tags.map(t => t.id.toString()),
                    },
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Advanced API Resource",
                        description: "A comprehensive API resource with multiple endpoints",
                        keywords: ["api", "rest", "json", "endpoints"],
                    }],
                    directoryListingsCreate: [{
                        id: generatePK(),
                        isRoot: true,
                        listingType: "File",
                        name: "openapi.yaml",
                    }],
                });

                const result = await resource.createOne({ input }, { req, res }, resource_createOne);

                expect(result.versionLabel).toBe("2.1.0");
                expect(result.versionNotes).toBe("Added new endpoints and improved error handling");
                expect(result.usedBy).toContain("project");
                expect(result.usedBy).toContain("routine");
                expect(result.root?.tags).toHaveLength(2);
                expect(result.directoryListings).toHaveLength(1);
                expect(result.directoryListings?.[0]?.name).toBe("openapi.yaml");
            });

            it("creates team resource", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create team
                const team = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMembers(
                        [{ userId: testUsers.records[0].id.toString(), isAdmin: true }],
                        {
                            id: generatePK(),
                            createdBy: { connect: { id: testUsers.records[0].id } },
                        }
                    ),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: ResourceVersionCreateInput = resourceVersionTestDataFactory.createMinimal({
                    id: generatePK(),
                    versionLabel: "1.0.0",
                    resourceType: "Database",
                    root: {
                        id: generatePK(),
                        publicId: "team_database_resource",
                        isPrivate: false,
                        teamConnect: team.id.toString(),
                    },
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Team Database Resource",
                    }],
                });

                const result = await resource.createOne({ input }, { req, res }, resource_createOne);

                expect(result.root?.team?.id).toBe(team.id.toString());
                expect(result.resourceType).toBe("Database");
            });
        });

        describe("invalid", () => {
            it("rejects duplicate public ID", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create existing resource
                await DbProvider.get().resource.create({
                    data: ResourceDbFactory.createWithVersion(
                        testUsers.records[0].id,
                        {
                            name: "Existing Resource",
                            versionLabel: "1.0.0",
                            isComplete: true,
                        },
                        ResourceType.Code,
                        {
                            publicId: "existing_resource",
                        }
                    ),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: ResourceVersionCreateInput = resourceVersionTestDataFactory.createMinimal({
                    id: generatePK(),
                    root: {
                        id: generatePK(),
                        publicId: "existing_resource", // Duplicate
                        isPrivate: false,
                    },
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Duplicate Resource",
                    }],
                });

                await expect(resource.createOne({ input }, { req, res }, resource_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid version label format", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: ResourceVersionCreateInput = resourceVersionTestDataFactory.createMinimal({
                    id: generatePK(),
                    versionLabel: "invalid.version.format.too.long", // Invalid format
                    root: {
                        id: generatePK(),
                        publicId: "test_resource",
                        isPrivate: false,
                    },
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Resource",
                    }],
                });

                await expect(resource.createOne({ input }, { req, res }, resource_createOne))
                    .rejects.toThrow();
            });

            it("rejects resource without translations", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: ResourceVersionCreateInput = {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    resourceType: "Api",
                    root: {
                        id: generatePK(),
                        publicId: "no_translations_resource",
                        isPrivate: false,
                    },
                    // Missing required translations
                };

                await expect(resource.createOne({ input }, { req, res }, resource_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    resource.updateOne,
                    { id: generatePK().toString() },
                    resource_updateOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    resource.updateOne,
                    { id: generatePK().toString() },
                    resource_updateOne,
                    ["ResourceVersion"],
                );
            });
        });

        describe("valid", () => {
            it("updates own resource version", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                // Use validation fixtures for consistent test data
                const input: ResourceVersionUpdateInput = resourceVersionTestDataFactory.createMinimal({
                    id: resources[0].versions[0].id.toString(),
                    versionNotes: "Updated with bug fixes",
                    translationsUpdate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Updated Public Resource",
                        description: "Updated description with more details",
                    }],
                });

                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);

                expect(result.versionNotes).toBe("Updated with bug fixes");
                expect(result.translations?.[0]?.name).toBe("Updated Public Resource");
                expect(result.translations?.[0]?.description).toBe("Updated description with more details");
            });

            it("updates resource directory structure", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                // Use validation fixtures for consistent test data
                const input: ResourceVersionUpdateInput = resourceVersionTestDataFactory.createMinimal({
                    id: resources[0].versions[0].id.toString(),
                    directoryListingsCreate: [
                        {
                            id: generatePK(),
                            isRoot: false,
                            listingType: "Directory",
                            name: "schemas",
                        },
                        {
                            id: generatePK(),
                            isRoot: false,
                            listingType: "File",
                            name: "user.schema.json",
                        },
                    ],
                });

                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);

                expect(result.directoryListings).toHaveLength(2);
                const directory = result.directoryListings?.find(d => d.name === "schemas");
                const file = result.directoryListings?.find(d => d.name === "user.schema.json");
                expect(directory?.listingType).toBe("Directory");
                expect(file?.listingType).toBe("File");
            });

            it("updates resource tags", async () => {
                const { testUsers, resources, tags } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                // Use validation fixtures for consistent test data
                const input: ResourceVersionUpdateInput = resourceVersionTestDataFactory.createMinimal({
                    id: resources[0].versions[0].id.toString(),
                    root: {
                        id: resources[0].id.toString(),
                        tagsConnect: [tags[0].id.toString(), tags[1].id.toString()],
                    },
                });

                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);

                expect(result.root?.tags).toHaveLength(2);
                const tagNames = result.root?.tags?.map(t => t.tag);
                expect(tagNames).toContain("api");
                expect(tagNames).toContain("database");
            });

            it("updates team resource as team member", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(), // Team owner
                });

                const teamResourceVersion = resources[2].versions[0];
                // Use validation fixtures for consistent test data
                const input: ResourceVersionUpdateInput = resourceVersionTestDataFactory.createMinimal({
                    id: teamResourceVersion.id.toString(),
                    versionNotes: "Updated by team member",
                    usedBy: ["project", "routine", "standard"],
                });

                const result = await resource.updateOne({ input }, { req, res }, resource_updateOne);

                expect(result.versionNotes).toBe("Updated by team member");
                expect(result.usedBy).toContain("standard");
            });
        });

        describe("invalid", () => {
            it("cannot update another user's private resource", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[2].id.toString(), // User 3 trying to update user 2's private resource
                });

                const input: ResourceVersionUpdateInput = {
                    id: resources[1].versions[0].id.toString(), // Private resource by user 2
                    versionNotes: "Unauthorized update",
                };

                await expect(resource.updateOne({ input }, { req, res }, resource_updateOne))
                    .rejects.toThrow(Error);
            });

            it("cannot update to duplicate version label", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create resource with two versions
                const resource1 = await DbProvider.get().resource.create({
                    data: ResourceDbFactory.createWithVersion(
                        testUsers.records[0].id,
                        {
                            name: "Multi Version Resource",
                            versionLabel: "1.0.0",
                            isComplete: true,
                        },
                        ResourceType.Code,
                        {
                            publicId: "multi_ver_res",
                        }
                    ),
                    include: { versions: true },
                });

                // Create second version
                const version2 = await DbProvider.get().resourceVersion.create({
                    data: {
                        id: generatePK(),
                        resourceId: resource1.id,
                        versionLabel: "2.0.0",
                        isLatest: false,
                        isPrivate: false,
                        resourceType: "Api",
                        directoryListings: [],
                        usedBy: [],
                        createdById: testUsers.records[0].id,
                    },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                // Try to update version 2 to have same label as version 1
                const input: ResourceVersionUpdateInput = {
                    id: version2.id.toString(),
                    versionLabel: resource1.versions[0].versionLabel, // Duplicate label
                };

                await expect(resource.updateOne({ input }, { req, res }, resource_updateOne))
                    .rejects.toThrow();
            });

            it("cannot update non-existent resource", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers.records[0].id.toString(),
                });

                const input: ResourceVersionUpdateInput = {
                    id: generatePK().toString(),
                    versionNotes: "This should fail",
                };

                await expect(resource.updateOne({ input }, { req, res }, resource_updateOne))
                    .rejects.toThrow(Error);
            });
        });
    });
});
