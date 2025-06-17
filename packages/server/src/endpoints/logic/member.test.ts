import { type FindByIdInput, type MemberSearchInput, type MemberUpdateInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
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
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing  

describe("EndpointsMember", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.team.deleteMany();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns member by id for team member", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[0].id.toString()
            });
            const input: FindByIdInput = { id: memberData.members[0].id };
            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[0].id);
            expect(result.role).toBe("Owner");
        });

        it("returns member by id for another team member", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[1].id.toString()
            });
            const input: FindByIdInput = { id: memberData.members[1].id };
            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[1].id);
            expect(result.role).toBe("Member");
        });

        it("returns member by id with API key public read", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[0].id.toString()
            });
            const input: FindByIdInput = { id: memberData.members[0].id };
            const result = await member.findOne({ input }, { req, res }, member_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(memberData.members[0].id);
        });

        it("throws error for not authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
            const input: FindByIdInput = { id: memberData.members[0].id };

            await expect(async () => {
                await member.findOne({ input }, { req, res }, member_findOne);
            }).rejects.toThrow();
        });
    });

    describe("findMany", () => {
        it("returns members for a team", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[0].id.toString()
            });
            const input: MemberSearchInput = {
                teamIds: [team.id],
                take: 10,
            };
            const result = await member.findMany({ input }, { req, res }, member_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBe(3); // Owner, Member, Admin

            // Check roles
            const roles = result.edges.map((edge: any) => edge.node.role);
            expect(roles).toContain("Owner");
            expect(roles).toContain("Member");
            expect(roles).toContain("Admin");
        });

        it("returns members for non-member with public team", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[3].id.toString()
            });
            const input: MemberSearchInput = {
                teamIds: [team.id],
                take: 10,
            };
            const result = await member.findMany({ input }, { req, res }, member_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBeGreaterThan(0);
        });

        it("returns members with API key public read", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[0].id.toString()
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
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[0].id.toString()
            });

            // Use validation fixtures for update
            const input: MemberUpdateInput = {
                id: memberData.members[1].id, // Regular member
                role: "Admin",
            };

            const result = await member.updateOne({ input }, { req, res }, member_updateOne);
            expect(result).not.toBeNull();
            expect(result.role).toBe("Admin");
        });

        it("team admin can update member permissions", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[2].id.toString() // Admin user
            });

            const input: MemberUpdateInput = {
                id: memberData.members[1].id,
                permissions: ["CanComment", "CanUpdate"],
            };

            const result = await member.updateOne({ input }, { req, res }, member_updateOne);
            expect(result).not.toBeNull();
            expect(result.permissions).toEqual(["CanComment", "CanUpdate"]);
        });

        it("regular member cannot update other members", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: testUsers[1].id.toString() // Regular member
            });

            const input: MemberUpdateInput = {
                id: memberData.members[2].id, // Try to update admin
                role: "Member",
            };

            await expect(async () => {
                await member.updateOne({ input }, { req, res }, member_updateOne);
            }).rejects.toThrow();
        });

        it("throws error for not authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 4, { withAuth: true });

            // Create a team
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    isPrivate: false,
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Test Team",
                            bio: "A team for testing",
                        },
                    },
                    owner: { connect: { id: testUsers[0].id } },
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
                id: memberData.members[0].id,
                role: "Member",
            };

            await expect(async () => {
                await member.updateOne({ input }, { req, res }, member_updateOne);
            }).rejects.toThrow();
        });
    });
});