import { type FindByPublicIdInput, type TeamCreateInput, type TeamSearchInput, type TeamUpdateInput, teamTestDataFactory, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession, mockApiSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { testEndpointRequiresAuth, testEndpointRequiresApiKeyReadPermissions, testEndpointRequiresApiKeyWritePermissions } from "../../__test/endpoints.js";
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
import { seedTestUsers, UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
import { TeamDbFactory } from "../../__test/fixtures/db/teamFixtures.js";

describe("EndpointsTeam", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.team_member.deleteMany();
        await prisma.team.deleteMany();
        await prisma.tag.deleteMany();
        await prisma.user.deleteMany();
        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    const createTestData = async () => {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
        
        // Create test tags
        const tags = await Promise.all([
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: "javascript" } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: "opensource" } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: "collaboration" } }),
        ]);

        // Create test teams
        const teams = await Promise.all([
            // Public team owned by user 1
            DbProvider.get().team.create({
                data: TeamDbFactory.createWithMember({
                    id: generatePK(),
                    handle: "public-team",
                    publicId: "public_team_123",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    createdById: testUsers[0].id,
                    members: [{
                        userId: testUsers[0].id,
                        permissions: "[\"owner\"]",
                    }],
                }),
                include: { members: true },
            }),
            // Private team owned by user 2
            DbProvider.get().team.create({
                data: TeamDbFactory.createWithMember({
                    id: generatePK(),
                    handle: "private-team",
                    publicId: "private_team_456",
                    isPrivate: true,
                    isOpenToNewMembers: false,
                    createdById: testUsers[1].id,
                    members: [{
                        userId: testUsers[1].id,
                        permissions: "[\"owner\"]",
                    }],
                }),
                include: { members: true },
            }),
            // Team with multiple members
            DbProvider.get().team.create({
                data: TeamDbFactory.createWithMember({
                    id: generatePK(),
                    handle: "multi-member-team",
                    publicId: "multi_member_789",
                    isPrivate: false,
                    createdById: testUsers[0].id,
                    members: [
                        {
                            userId: testUsers[0].id,
                            permissions: "[\"owner\"]",
                        },
                        {
                            userId: testUsers[1].id,
                            permissions: "[\"editChildObjects\", \"viewChildObjects\"]",
                        },
                    ],
                }),
                include: { members: true },
            }),
        ]);

        return { testUsers, tags, teams };
    };

    describe("findOne", () => {
        describe("authentication", () => {
            it("allows unauthenticated access to public teams", async () => {
                const { teams } = await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: teams[0].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[0].id.toString());
                expect(result.handle).toBe("public-team");
            });

            it("supports API key with read permissions", async () => {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockApiSession({
                    userId: testUsers[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockReadPublicPermissions(["Team"]),
                });

                const input: FindByPublicIdInput = { publicId: teams[0].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[0].id.toString());
            });
        });

        describe("valid", () => {
            it("returns public team details", async () => {
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

            it("returns private team details to member", async () => {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: teams[1].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teams[1].id.toString());
                expect(result.handle).toBe("private-team");
                expect(result.isPrivate).toBe(true);
            });

            it("returns team with multiple members", async () => {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: teams[2].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).not.toBeNull();
                expect(result.members?.length).toBe(2);
                
                const ownerMember = result.members?.find(m => 
                    m.user?.id === testUsers[0].id.toString(),
                );
                expect(ownerMember?.permissions).toContain("owner");
            });
        });

        describe("invalid", () => {
            it("returns null for private team when not a member", async () => {
                const { testUsers, teams } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[2].id.toString(), // User 3 is not a member
                });

                const input: FindByPublicIdInput = { publicId: teams[1].publicId };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent team", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByPublicIdInput = { publicId: "non_existent_team" };
                const result = await team.findOne({ input }, { req, res }, team_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            it("allows unauthenticated access", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result).toBeDefined();
                expect(result.results).toBeDefined();
                // Should only see public teams
                expect(result.results.every(t => !t.isPrivate)).toBe(true);
            });

            it("supports API key with read permissions", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockApiSession({
                    userId: testUsers[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockReadPublicPermissions(["Team"]),
                });

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toBeDefined();
            });
        });

        describe("valid", () => {
            it("returns public teams for unauthenticated users", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(2); // 2 public teams
                expect(result.results.every(t => !t.isPrivate)).toBe(true);
                expect(result.totalCount).toBe(2);
            });

            it("returns all accessible teams for authenticated user", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(), // Member of private team
                });

                const input: TeamSearchInput = {};
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(3); // All teams accessible
                expect(result.totalCount).toBe(3);
            });

            it("filters teams by search query", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    searchString: "public",
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].handle).toBe("public-team");
            });

            it("filters teams by member count", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    minMembers: 2,
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].handle).toBe("multi-member-team");
            });

            it("sorts teams by member count", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: TeamSearchInput = {
                    sortBy: "MembersDesc",
                };
                const result = await team.findMany({ input }, { req, res }, team_findMany);

                // Multi-member team should be first
                expect(result.results[0].handle).toBe("multi-member-team");
            });

            it("paginates results", async () => {
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

    describe("createOne", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    team.createOne,
                    teamTestDataFactory.createMinimal(),
                    team_createOne,
                );
            });

            it("requires API key with write permissions", async () => {
                await testEndpointRequiresApiKeyWritePermissions(
                    team.createOne,
                    teamTestDataFactory.createMinimal(),
                    team_createOne,
                    ["Team"],
                );
            });
        });

        describe("valid", () => {
            it("creates minimal team", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK(),
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Team",
                    }],
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

            it("creates complete team with all fields", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                // Create tags for the team
                const tags = await Promise.all([
                    DbProvider.get().tag.create({ data: { id: generatePK(), tag: "javascript" } }),
                    DbProvider.get().tag.create({ data: { id: generatePK(), tag: "opensource" } }),
                ]);

                const input: TeamCreateInput = teamTestDataFactory.createComplete({
                    id: generatePK(),
                    handle: "awesome-dev-team",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    bannerImage: "banner.jpg",
                    profileImage: "profile.png",
                    tagsConnect: tags.map(t => t.id.toString()),
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Awesome Dev Team",
                        bio: "A team of awesome developers",
                    }],
                });

                const result = await team.createOne({ input }, { req, res }, team_createOne);

                expect(result.handle).toBe("awesome-dev-team");
                expect(result.isPrivate).toBe(false);
                expect(result.isOpenToNewMembers).toBe(true);
                expect(result.bannerImage).toBe("banner.jpg");
                expect(result.profileImage).toBe("profile.png");
                expect(result.tags).toHaveLength(2);
                expect(result.translations?.[0]?.bio).toBe("A team of awesome developers");
            });

            it("creates team with custom config", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK(),
                    config: {
                        maxMembers: 50,
                        allowPublicProjects: true,
                        requireInviteApproval: false,
                    },
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Config Test Team",
                    }],
                });

                const result = await team.createOne({ input }, { req, res }, team_createOne);

                expect(result.config?.maxMembers).toBe(50);
                expect(result.config?.allowPublicProjects).toBe(true);
                expect(result.config?.requireInviteApproval).toBe(false);
            });
        });

        describe("invalid", () => {
            it("rejects duplicate handle", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create existing team
                await DbProvider.get().team.create({
                    data: TeamDbFactory.createMinimal({
                        handle: "existing-team",
                        createdById: testUser[0].id,
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK(),
                    handle: "existing-team", // Duplicate handle
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Duplicate Team",
                    }],
                });

                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid handle format", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamCreateInput = teamTestDataFactory.createMinimal({
                    id: generatePK(),
                    handle: "Invalid Handle!", // Invalid characters
                    translationsCreate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Invalid Team",
                    }],
                });

                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects.toThrow();
            });

            it("rejects missing required fields", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamCreateInput = {
                    id: generatePK(),
                    // Missing required translations
                };

                await expect(team.createOne({ input }, { req, res }, team_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    team.updateOne,
                    { id: generatePK() },
                    team_updateOne,
                );
            });

            it("requires API key with write permissions", async () => {
                await testEndpointRequiresApiKeyWritePermissions(
                    team.updateOne,
                    { id: generatePK() },
                    team_updateOne,
                    ["Team"],
                );
            });
        });

        describe("valid", () => {
            it("updates team as owner", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create team
                const existingTeam = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMember({
                        id: generatePK(),
                        handle: "original-team",
                        createdById: testUser[0].id,
                        members: [{
                            userId: testUser[0].id,
                            permissions: "[\"owner\"]",
                        }],
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    handle: "updated-team",
                    isOpenToNewMembers: false,
                    translationsUpdate: [{
                        id: generatePK(),
                        language: "en",
                        name: "Updated Team Name",
                    }],
                };

                const result = await team.updateOne({ input }, { req, res }, team_updateOne);

                expect(result.handle).toBe("updated-team");
                expect(result.isOpenToNewMembers).toBe(false);
                expect(result.translations?.[0]?.name).toBe("Updated Team Name");
            });

            it("updates team member permissions", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create team with multiple members
                const existingTeam = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMember({
                        id: generatePK(),
                        createdById: testUsers[0].id,
                        members: [
                            {
                                userId: testUsers[0].id,
                                permissions: "[\"owner\"]",
                            },
                            {
                                userId: testUsers[1].id,
                                permissions: "[\"viewChildObjects\"]",
                            },
                        ],
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    membersUpdate: [{
                        id: generatePK(),
                        userId: testUsers[1].id.toString(),
                        permissions: ["editChildObjects", "viewChildObjects"],
                    }],
                };

                const result = await team.updateOne({ input }, { req, res }, team_updateOne);

                const updatedMember = result.members?.find(m => 
                    m.user?.id === testUsers[1].id.toString(),
                );
                expect(updatedMember?.permissions).toContain("editChildObjects");
            });

            it("adds new team members", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create team with single member
                const existingTeam = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMember({
                        id: generatePK(),
                        createdById: testUsers[0].id,
                        members: [{
                            userId: testUsers[0].id,
                            permissions: "[\"owner\"]",
                        }],
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    membersCreate: [{
                        id: generatePK(),
                        userId: testUsers[1].id.toString(),
                        permissions: ["viewChildObjects"],
                    }],
                };

                const result = await team.updateOne({ input }, { req, res }, team_updateOne);

                expect(result.members).toHaveLength(2);
                const newMember = result.members?.find(m => 
                    m.user?.id === testUsers[1].id.toString(),
                );
                expect(newMember).toBeDefined();
                expect(newMember?.permissions).toContain("viewChildObjects");
            });
        });

        describe("invalid", () => {
            it("cannot update team without permission", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create team owned by user 1
                const existingTeam = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMember({
                        id: generatePK(),
                        createdById: testUsers[0].id,
                        members: [{
                            userId: testUsers[0].id,
                            permissions: "[\"owner\"]",
                        }],
                    }),
                });

                // Try to update as user 2
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    handle: "hacked-team",
                };

                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("cannot update to existing handle", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create two teams
                const team1 = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMember({
                        handle: "team-one",
                        createdById: testUser[0].id,
                        members: [{
                            userId: testUser[0].id,
                            permissions: "[\"owner\"]",
                        }],
                    }),
                });

                await DbProvider.get().team.create({
                    data: TeamDbFactory.createMinimal({
                        handle: "team-two",
                        createdById: testUser[0].id,
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamUpdateInput = {
                    id: team1.id.toString(),
                    handle: "team-two", // Existing handle
                };

                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects.toThrow();
            });

            it("cannot remove last owner", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create team with single owner
                const existingTeam = await DbProvider.get().team.create({
                    data: TeamDbFactory.createWithMember({
                        id: generatePK(),
                        createdById: testUser[0].id,
                        members: [{
                            id: generatePK(),
                            userId: testUser[0].id,
                            permissions: "[\"owner\"]",
                        }],
                    }),
                    include: { members: true },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: TeamUpdateInput = {
                    id: existingTeam.id.toString(),
                    membersDelete: [existingTeam.members[0].id.toString()],
                };

                await expect(team.updateOne({ input }, { req, res }, team_updateOne))
                    .rejects.toThrow(CustomError);
            });
        });
    });
});
