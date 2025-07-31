import { type FindByPublicIdInput, type TeamCreateInput, type TeamSearchInput, type TeamUpdateInput, generatePK, generatePublicId } from "@vrooli/shared";
import { teamTestDataFactory } from "@vrooli/shared/test-fixtures/api-inputs";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
// Removed deprecated test helpers - using direct endpoint testing instead
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { team_createOne } from "../generated/team_createOne.js";
import { team_findMany } from "../generated/team_findMany.js";
import { team_findOne } from "../generated/team_findOne.js";
import { team_updateOne } from "../generated/team_updateOne.js";
import { team } from "./team.js";
// Import database fixtures
import { TeamDbFactory } from "../../__test/fixtures/db/teamFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsTeam", function describeEndpointsTeam() {
    beforeAll(async function setupBeforeAll() {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(function mockError() { return logger; });
        vi.spyOn(logger, "info").mockImplementation(function mockInfo() { return logger; });
        vi.spyOn(logger, "warning").mockImplementation(function mockWarning() { return logger; });
    });

    afterEach(async () => {
        // Perform cleanup using dependency-ordered cleanup helpers
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

    beforeEach(async function setupBeforeEach() {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.team(DbProvider.get());

        // Clean teams before users to avoid FK constraints
        await DbProvider.get().team.deleteMany();

        // Clean user-related data (all potential relationships created by seedTestUsers)
        await DbProvider.get().session.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().email.deleteMany();
        await DbProvider.get().phone.deleteMany();
        await DbProvider.get().user_translation.deleteMany();
        await DbProvider.get().push_device.deleteMany();
        await DbProvider.get().wallet.deleteMany();
        await DbProvider.get().api_key.deleteMany();
        await DbProvider.get().credit_account.deleteMany();
        await DbProvider.get().award.deleteMany();

        // Finally clean users
        await DbProvider.get().user.deleteMany();

        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async function cleanupAfterAll() {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create authenticated session with user data
    function createAuthSessionData(user: any) {
        return {
            ...loggedInUserNoPremiumData(),
            id: user.id.toString(),
            handle: user.handle || `testuser_${user.id}`,
            name: user.name || `Test User ${user.id}`,
            publicId: user.publicId,
        };
    }

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const testUsersResult = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
        const testUsers = testUsersResult.records;

        // Create test tags with unique names to avoid conflicts
        const uniqueSuffix = Date.now().toString();
        const tags = await Promise.all([
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `javascript_${uniqueSuffix}` } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `opensource_${uniqueSuffix}` } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `collaboration_${uniqueSuffix}` } }),
        ]);

        // Create test teams using Option 2 approach
        const teams = await Promise.all([
            // Public team owned by user 1
            DbProvider.get().team.create({
                data: TeamDbFactory.createMinimal({
                    handle: "public-team",
                    publicId: "public_team_123",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                }),
                include: { members: true },
            }).then(async function handleTeamCreation(team) {
                // Add member separately
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: team.id,
                        userId: testUsers[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });
                return DbProvider.get().team.findUnique({
                    where: { id: team.id },
                    include: { members: true },
                }).then(function verifyTeamExists(result) {
                    if (!result) throw new Error("Team not found after creation");
                    return result;
                });
            }),
            // Private team owned by user 2
            DbProvider.get().team.create({
                data: TeamDbFactory.createMinimal({
                    handle: "private-team",
                    publicId: "private_team_456",
                    isPrivate: true,
                    isOpenToNewMembers: false,
                    createdBy: { connect: { id: testUsers[1].id } },
                }),
                include: { members: true },
            }).then(async function handleTeamCreation(team) {
                // Add member separately
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: team.id,
                        userId: testUsers[1].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });
                return DbProvider.get().team.findUnique({
                    where: { id: team.id },
                    include: { members: true },
                }).then(function verifyTeamExists(result) {
                    if (!result) throw new Error("Team not found after creation");
                    return result;
                });
            }),
            // Team with multiple members
            DbProvider.get().team.create({
                data: TeamDbFactory.createMinimal({
                    handle: "multi-member-team",
                    publicId: "multi_member_789",
                    isPrivate: false,
                    createdBy: { connect: { id: testUsers[0].id } },
                }),
                include: { members: true },
            }).then(async function handleMultiMemberTeam(team) {
                // Add members separately
                await DbProvider.get().member.createMany({
                    data: [
                        {
                            id: generatePK(),
                            publicId: generatePublicId(),
                            teamId: team.id,
                            userId: testUsers[0].id,
                            isAdmin: true,
                            permissions: "[\"owner\"]",
                        },
                        {
                            id: generatePK(),
                            publicId: generatePublicId(),
                            teamId: team.id,
                            userId: testUsers[1].id,
                            isAdmin: false,
                            permissions: "[\"editChildObjects\", \"viewChildObjects\"]",
                        },
                    ],
                });
                return DbProvider.get().team.findUnique({
                    where: { id: team.id },
                    include: { members: true },
                }).then(function verifyTeamExists(result) {
                    if (!result) throw new Error("Team not found after creation");
                    return result;
                });
            }),
        ]);

        return { testUsers, tags, teams };
    }

    describe("findOne", function describeFindOne() {
        describe("authentication", function describeAuthentication() {
            it("allows unauthenticated access to public teams", async function testUnauthenticatedAccess() {
                const { teams } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: teams[0].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[0].id.toString());
                expect(result.handle).toBe("public-team");
            });

            it("supports API key with read permissions", async function testApiKeyReadPermissions() {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockApiSession(
                    "test-api-key-" + generatePK().toString(),
                    mockReadPublicPermissions(),
                    { id: testUsers[0].id.toString(), emails: [{ emailAddress: "test@example.com" }], handle: "testuser" },
                );

                const input: FindByPublicIdInput = { publicId: teams[0].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[0].id.toString());
            });
        });

        describe("valid", function describeValid() {
            it("returns public team details", async function testPublicTeamDetails() {
                const { teams } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: teams[0].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[0].id.toString());
                expect(result.handle).toBe("public-team");
                expect(result.isPrivate).toBe(false);
                expect(result.isOpenToNewMembers).toBe(true);
                expect(result.members).toBeDefined();
                expect(result.members?.length).toBe(1);
            });

            it("returns private team details to member", async function testPrivateTeamMemberAccess() {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[1]));

                const input: FindByPublicIdInput = { publicId: teams[1].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[1].id.toString());
                expect(result.handle).toBe("private-team");
                expect(result.isPrivate).toBe(true);
            });

            it("returns team with multiple members", async function testMultipleMembersTeam() {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[0]));

                const input: FindByPublicIdInput = { publicId: teams[2].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.members?.length).toBe(2);

                const ownerMember = result.members?.find(function isOwner(m) {
                    return m.user?.id === testUsers[0].id.toString();
                });
                expect(ownerMember?.permissions).toContain("owner");
            });
        });

        describe("invalid", function describeInvalid() {
            it("returns null for private team when not a member", async function testPrivateTeamNonMemberAccess() {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[2]));

                const input: FindByPublicIdInput = { publicId: teams[1].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent team", async function testNonExistentTeam() {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: "non_existent_team" };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", function describeFindMany() {
        describe("authentication", function describeFindManyAuthentication() {
            it("allows unauthenticated access", async function testUnauthenticatedFindMany() {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result).toBeDefined();
                expect(result.results).toBeDefined();
                // Should only see public teams
                expect(result.results.every(function isPublic(t) { return !t.isPrivate; })).toBe(true);
            });

            it("supports API key with read permissions", async function testApiKeyFindMany() {
                const { testUsers } = await createTestData();
                const { req, res } = await mockApiSession(
                    "test-api-key-" + generatePK().toString(),
                    mockReadPublicPermissions(),
                    { id: testUsers[0].id.toString(), emails: [{ emailAddress: "test@example.com" }], handle: "testuser" },
                );

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toBeDefined();
            });
        });

        describe("valid", function describeFindManyValid() {
            it("returns public teams for unauthenticated users", async function testPublicTeamsUnauthenticated() {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(2); // 2 public teams
                expect(result.results.every(function isPublic(t) { return !t.isPrivate; })).toBe(true);
                expect(result.totalCount).toBe(2);
            });

            it("returns all accessible teams for authenticated user", async function testAllAccessibleTeams() {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[1]));

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(3); // All teams accessible
                expect(result.totalCount).toBe(3);
            });

            it("filters teams by search query", async function testSearchFilter() {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    searchString: "public",
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].handle).toBe("public-team");
            });

            it("filters teams by member count", async function testMemberCountFilter() {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    minMembers: 2,
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].handle).toBe("multi-member-team");
            });

            it("sorts teams by member count", async function testMemberCountSort() {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    sortBy: "MembersDesc",
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                // Multi-member team should be first
                expect(result.results[0].handle).toBe("multi-member-team");
            });

            it("paginates results", async function testPagination() {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    take: 1,
                    skip: 0,
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.totalCount).toBe(2);
            });
        });
    });

    describe("createOne", function describeCreateOne() {
        describe("authentication", function describeCreateAuthentication() {
            it("not logged in", async function testNotLoggedIn() {
                const { req, res } = await mockLoggedOutSession();
                const input = teamTestDataFactory.createMinimal();
                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects
                    .toThrow(expect.objectContaining({
                        code: "0002",
                    }));
            });

            it("API key - no write permissions", async function testApiKeyNoWritePermissions() {
                const userData = loggedInUserNoPremiumData();
                const { req, res } = await mockApiSession(
                    "test-api-key",
                    mockReadPublicPermissions(), // Only read permissions, not write
                    userData,
                );
                const input = teamTestDataFactory.createMinimal();
                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects
                    .toThrow(expect.objectContaining({
                        code: "0002",
                    }));
            });
        });

        describe("valid", function describeValid() {
            it("creates minimal team", async function testCreateMinimalTeam() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                });

                const result = await team.createOne({ input }, { req, res }, team_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Team");
                expect(result.id).toBe(input.id);
                expect(result.translations?.[0]?.name).toBe("Test Team");
                expect(result.members).toHaveLength(1);
                expect(result.members?.[0]?.user?.id).toBe(testUser[0].id.toString());
                expect(result.members?.[0]?.permissions).toContain("owner");

                // Verify in database
                const createdTeam = await DbProvider.get().team.findUnique({
                    where: { id: BigInt(input.id) },
                    include: { members: true },
                });
                expect(createdTeam).toBeDefined();
                expect(createdTeam?.members).toHaveLength(1);
            });

            it("creates complete team with all fields", async function testCreateCompleteTeam() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                // Create tags for the team with unique names
                const uniqueSuffix = Date.now().toString();
                const tags = await Promise.all([
                    DbProvider.get().tag.create({ data: { id: generatePK(), tag: `javascript_${uniqueSuffix}` } }),
                    DbProvider.get().tag.create({ data: { id: generatePK(), tag: `opensource_${uniqueSuffix}` } }),
                ]);

                const input: TeamCreateInput = teamTestDataFactory.createComplete({
                    id: generatePK().toString(),
                    handle: "awesome-dev-team",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    tagsConnect: tags.map(function tagToId(t) { return t.id.toString(); }),
                });

                const result = await team.createOne({ input }, { req, res }, team_createOne);

                expect(result.handle).toBe("awesome-dev-team");
                expect(result.isPrivate).toBe(false);
                expect(result.isOpenToNewMembers).toBe(true);
                expect(result.tags).toHaveLength(2);
                expect(result.translations?.[0]?.bio).toBe("We are building amazing things together.");
            });

            it("creates team with custom config", async function testCreateTeamWithConfig() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    config: {
                        __version: "1.0",
                        resources: [],
                        deploymentType: "development",
                        goal: "Test team goal",
                        businessPrompt: "Test business prompt",
                        resourceQuota: { gpuPercentage: 10, ramGB: 8, cpuCores: 2, storageGB: 50 },
                        targetProfitPerMonth: "0",
                        stats: {
                            totalInstances: 0,
                            totalProfit: "0",
                            totalCosts: "0",
                            averageKPIs: {},
                            activeInstances: 0,
                            lastUpdated: Date.now(),
                        },
                        structure: {
                            type: "MOISE+",
                            version: "1.0",
                            content: "structure TestTeam { group main { role member cardinality 1..50 } }",
                        },
                    },
                });

                const result = await team.createOne({ input }, { req, res }, team_createOne);

                expect(result.config).toBeDefined();
                expect(result.config?.structure?.type).toBe("MOISE+");
                expect(result.config?.structure?.content).toContain("TestTeam");
            });
        });

        describe("invalid", function describeInvalid() {
            it("rejects duplicate handle", async function testDuplicateHandle() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;

                // Create existing team
                await DbProvider.get().team.create({
                    data: TeamDbFactory.createMinimal({
                        handle: "existing-team",
                        createdBy: { connect: { id: testUser[0].id } },
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    handle: "existing-team", // Duplicate handle
                });

                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid handle format", async function testInvalidHandleFormat() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    handle: "Invalid Handle!", // Invalid characters
                });

                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects.toThrow();
            });

            it("rejects missing required fields", async function testMissingRequiredFields() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamCreateInput = {
                    id: generatePK().toString(),
                    // Missing required translations
                } as TeamCreateInput;

                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", function describeUpdateOne() {
        describe("authentication", function describeUpdateAuthentication() {
            it("not logged in", async function testUpdateNotLoggedIn() {
                const { req, res } = await mockLoggedOutSession();
                const input: TeamUpdateInput = { id: generatePK().toString() };
                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects
                    .toThrow(expect.objectContaining({
                        code: "0002",
                    }));
            });

            it("API key - no write permissions", async function testUpdateApiKeyNoWritePermissions() {
                const userData = loggedInUserNoPremiumData();
                const { req, res } = await mockApiSession(
                    "test-api-key",
                    mockReadPublicPermissions(), // Only read permissions, not write
                    userData,
                );
                const input: TeamUpdateInput = { id: generatePK().toString() };
                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects
                    .toThrow(expect.objectContaining({
                        code: "0002",
                    }));
            });
        });

        describe("valid", function describeValid() {
            it("updates team as owner", async function testUpdateTeamAsOwner() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;

                // Create team using factory
                const teamData = TeamDbFactory.createMinimal({
                    handle: "original-team",
                    createdBy: { connect: { id: testUser[0].id } },
                });
                const existingTeam = await DbProvider.get().team.create({
                    data: teamData,
                });
                // Add member separately
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: existingTeam.id,
                        userId: testUser[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });

                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    handle: "updated-team",
                    isOpenToNewMembers: false,
                    translationsUpdate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Updated Team Name",
                    }],
                };

                const result = await team.updateOne({ input }, { req, res }, team_updateOne);

                expect(result.handle).toBe("updated-team");
                expect(result.isOpenToNewMembers).toBe(false);
                expect(result.translations?.[0]?.name).toBe("Updated Team Name");
            });

            it("updates team basic info", async function testUpdateTeamInfo() {
                const testUsersResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUsers = testUsersResult.records;

                // Create team using factory
                const teamData = TeamDbFactory.createMinimal({
                    createdBy: { connect: { id: testUsers[0].id } },
                    handle: "original-handle",
                    isOpenToNewMembers: true,
                });
                const existingTeam = await DbProvider.get().team.create({
                    data: teamData,
                });
                // Add owner member
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: existingTeam.id,
                        userId: testUsers[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });

                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[0]));

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    handle: "updated-handle",
                    isOpenToNewMembers: false,
                    translationsUpdate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Updated Team Name",
                        bio: "Updated team description",
                    }],
                };

                const result = await team.updateOne({ input }, { req, res }, team_updateOne);

                expect(result.handle).toBe("updated-handle");
                expect(result.isOpenToNewMembers).toBe(false);
                expect(result.translations?.[0]?.name).toBe("Updated Team Name");
                expect(result.translations?.[0]?.bio).toBe("Updated team description");
            });

            it("creates member invites", async function testCreateMemberInvites() {
                const testUsersResult = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                const testUsers = testUsersResult.records;

                // Create team with single member using factory
                const teamData = TeamDbFactory.createMinimal({
                    createdBy: { connect: { id: testUsers[0].id } },
                });
                const existingTeam = await DbProvider.get().team.create({
                    data: teamData,
                });
                // Add owner member
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: existingTeam.id,
                        userId: testUsers[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });

                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[0]));

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    memberInvitesCreate: [{
                        id: generatePK().toString(),
                        message: "Please join our team!",
                        teamConnect: existingTeam.id.toString(),
                        userConnect: testUsers[1].id.toString(),
                    }],
                };

                const result = await team.updateOne({ input }, { req, res }, team_updateOne);

                expect(result).toBeDefined();
                expect(result.id).toBe(existingTeam.id.toString());
            });
        });

        describe("invalid", function describeInvalid() {
            it("cannot update team without permission", async function testUpdateWithoutPermission() {
                const testUsersResult = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                const testUsers = testUsersResult.records;

                // Create team owned by user 1 using factory
                const teamData = TeamDbFactory.createMinimal({
                    createdBy: { connect: { id: testUsers[0].id } },
                });
                const existingTeam = await DbProvider.get().team.create({
                    data: teamData,
                });
                // Add member separately
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: existingTeam.id,
                        userId: testUsers[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });

                // Try to update as user 2
                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUsers[1]));

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    handle: "hacked-team",
                };

                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("cannot update to existing handle", async function testUpdateToExistingHandle() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;

                // Create two teams using factory
                const teamData1 = TeamDbFactory.createMinimal({
                    handle: "team-one",
                    createdBy: { connect: { id: testUser[0].id } },
                });
                const team1 = await DbProvider.get().team.create({
                    data: teamData1,
                });
                // Add member separately
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: team1.id,
                        userId: testUser[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });

                await DbProvider.get().team.create({
                    data: TeamDbFactory.createMinimal({
                        handle: "team-two",
                        createdBy: { connect: { id: testUser[0].id } },
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamUpdateInput = {
                    id: team1.id.toString(),
                    handle: "team-two", // Existing handle
                };

                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects.toThrow();
            });

            it("cannot remove last owner", async function testCannotRemoveLastOwner() {
                const testUserResult = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const testUser = testUserResult.records;

                // Create team with single owner using factory
                const teamData = TeamDbFactory.createMinimal({
                    createdBy: { connect: { id: testUser[0].id } },
                });
                const existingTeam = await DbProvider.get().team.create({
                    data: teamData,
                });
                // Add member separately
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: existingTeam.id,
                        userId: testUser[0].id,
                        isAdmin: true,
                        permissions: "[\"owner\"]",
                    },
                });
                // Fetch team with members included
                const teamWithMembers = await DbProvider.get().team.findUnique({
                    where: { id: existingTeam.id },
                    include: { members: true },
                });

                if (!teamWithMembers || !teamWithMembers.members || teamWithMembers.members.length === 0) {
                    throw new Error("Team or members not found");
                }

                const { req, res } = await mockAuthenticatedSession(createAuthSessionData(testUser[0]));

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    membersDelete: [teamWithMembers.members[0].id.toString()],
                };

                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects.toThrow(CustomError);
            });
        });
    });
});
