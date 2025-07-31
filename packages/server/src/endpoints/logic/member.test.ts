import { type FindByPublicIdInput, type MemberSearchInput, type MemberUpdateInput, SEEDED_PUBLIC_IDS, generatePK, generatePublicId } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { member_findMany } from "../generated/member_findMany.js";
import { member_findOne } from "../generated/member_findOne.js";
import { member_updateOne } from "../generated/member_updateOne.js";
import { member } from "./member.js";
// Import database fixtures for seeding
import { seedTeamWithMembers } from "../../__test/fixtures/db/memberFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

// Import validation fixtures for API input testing  

describe("EndpointsMember", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);

        // Create admin user for system operations
        await DbProvider.get().user.upsert({
            where: { publicId: SEEDED_PUBLIC_IDS.Admin },
            update: {},
            create: {
                id: generatePK(),
                publicId: SEEDED_PUBLIC_IDS.Admin,
                handle: "admin",
                name: "Admin User",
                isPrivate: false,
            },
        });
    });

    afterEach(async () => {
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

        // Ensure admin user exists for each test
        await DbProvider.get().user.upsert({
            where: { publicId: SEEDED_PUBLIC_IDS.Admin },
            update: {},
            create: {
                id: generatePK(),
                publicId: SEEDED_PUBLIC_IDS.Admin,
                handle: "admin",
                name: "Admin User",
                isPrivate: false,
            },
        });
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns member by id for team member", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: FindByPublicIdInput = { publicId: memberData.members[0].publicId };
            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[0].id);
            expect(result.isAdmin).toBe(true);
            expect(result.permissions).toEqual(expect.objectContaining({
                manageTeam: true,
                manageMembers: true,
            }));
        });

        it("returns member by id for another team member", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id.toString(),
            });
            const input: FindByPublicIdInput = { publicId: memberData.members[1].publicId };
            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[1].id);
            expect(result.isAdmin).toBe(false);
            expect(result.permissions).toEqual({});
        });

        it("returns member by id with API key public read", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, {
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: FindByPublicIdInput = { publicId: memberData.members[0].publicId };
            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[0].id);
        });

        it("returns member data for public team even without authentication", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockLoggedOutSession();
            const input: FindByPublicIdInput = { publicId: memberData.members[0].publicId };

            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[0].id);
        });
    });

    describe("findMany", () => {
        it("returns members for a team", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: MemberSearchInput = {
                teamId: team.id.toString(),
                take: 10,
            };
            const result = await member.findMany({ input }, { req, res }, member_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBe(3); // Owner, Member, Admin

            // Check admin statuses and permissions
            const members = result.edges.map((edge: any) => edge.node);

            // Should have an owner (isAdmin: true with manageTeam permission)
            const owner = members.find((m: any) => m.isAdmin && m.permissions?.manageTeam);
            expect(owner).toBeDefined();

            // Should have a regular member (isAdmin: false)
            const regularMember = members.find((m: any) => !m.isAdmin);
            expect(regularMember).toBeDefined();

            // Should have an admin (isAdmin: true with manageMembers but not manageTeam)
            const admin = members.find((m: any) => m.isAdmin && m.permissions?.manageMembers && !m.permissions?.manageTeam);
            expect(admin).toBeDefined();
        });

        it("returns members for non-member with public team", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[3].id.toString(),
            });
            const input: MemberSearchInput = {
                teamId: team.id.toString(),
                take: 10,
            };
            const result = await member.findMany({ input }, { req, res }, member_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBeGreaterThan(0);
        });

        it("returns members with API key public read", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, {
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: MemberSearchInput = { take: 10 };
            const result = await member.findMany({ input }, { req, res }, member_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
        });
    });

    describe("updateOne", () => {
        it("team owner can update member role", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            // Use validation fixtures for update
            const input: MemberUpdateInput = {
                id: memberData.members[1].id.toString(), // Regular member
                isAdmin: true,
                permissions: JSON.stringify({ manageMembers: true }),
            };

            const result = await member.updateOne({ input }, { req, res }, member_updateOne);
            expect(result).not.toBeNull();
            expect(result.isAdmin).toBe(true);
            expect(result.permissions).toEqual(expect.objectContaining({
                manageMembers: true,
            }));
        });

        it("team admin can update member permissions", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[2].id.toString(), // Admin user
            });

            const input: MemberUpdateInput = {
                id: memberData.members[1].id.toString(),
                permissions: JSON.stringify(["CanComment", "CanUpdate"]),
            };

            const result = await member.updateOne({ input }, { req, res }, member_updateOne);
            expect(result).not.toBeNull();
            expect(result.permissions).toEqual(["CanComment", "CanUpdate"]);
        });

        it("regular member cannot update other members", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id.toString(), // Regular member
            });

            const input: MemberUpdateInput = {
                id: memberData.members[2].id.toString(), // Try to update admin
                isAdmin: false,
                permissions: JSON.stringify({}),
            };

            await expect(async () => {
                await member.updateOne({ input }, { req, res }, member_updateOne);
            }).rejects.toThrow();
        });

        it("throws error for not authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsersResult = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });
            const testUsers = testUsersResult.records;

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    handle: `team_test_${Date.now()}`,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                },
            });

            // Seed team members using database fixtures
            const memberData = await seedTeamWithMembers(DbProvider.get(), {
                teamId: team.id,
                ownerId: testUsers[0].id,
                memberIds: [testUsers[1].id],
                adminIds: [testUsers[2].id],
                withInvites: [{ userId: testUsers[3].id, willBeAdmin: false }],
            });

            const { req, res } = await mockLoggedOutSession();

            const input: MemberUpdateInput = {
                id: memberData.members[0].id.toString(),
                isAdmin: false,
                permissions: JSON.stringify({}),
            };

            await expect(async () => {
                await member.updateOne({ input }, { req, res }, member_updateOne);
            }).rejects.toThrow();
        });
    });
});
