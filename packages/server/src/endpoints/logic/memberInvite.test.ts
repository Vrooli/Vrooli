import { type FindByIdInput, type MemberInviteCreateInput, type MemberInviteSearchInput, type MemberInviteUpdateInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { memberInvite_acceptOne } from "../generated/memberInvite_acceptOne.js";
import { memberInvite_createOne } from "../generated/memberInvite_createOne.js";
import { memberInvite_declineOne } from "../generated/memberInvite_declineOne.js";
import { memberInvite_findMany } from "../generated/memberInvite_findMany.js";
import { memberInvite_findOne } from "../generated/memberInvite_findOne.js";
import { memberInvite_updateOne } from "../generated/memberInvite_updateOne.js";
import { memberInvite } from "./memberInvite.js";
// Import database fixtures for seeding
import { seedMemberInvites } from "../../__test/fixtures/db/memberInviteFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { memberInviteTestDataFactory } from "@vrooli/shared";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsMemberInvite", () => {
    let testUsers: any[];
    let team1: any;
    let team2: any;
    let publicTeam: any;
    let invite1: any;
    let invite2: any;
    let publicInvite: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user","user_auth","email","session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn('Test cleanup incomplete:', orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());
    });
            }
        } catch (error) {
            // If database is not initialized, skip cleanup
        }

        // Create teams for member invites
        team1 = await DbProvider.get().team.create({
            data: {
                id: testUsers[0].id + "team1",
                publicId: "team1-" + Math.random(),
                handle: "test-team-1",
                name: "Test Team 1",
                isPrivate: false,
                isOpenToAnyoneWithInvite: false,
                created_by: { connect: { id: testUsers[0].id } },
                members: {
                    create: [{
                        id: testUsers[0].id + "member1",
                        publicId: "member1-" + Math.random(),
                        user: { connect: { id: testUsers[0].id } },
                        permissions: JSON.stringify({ canEdit: true, canInvite: true }),
                    }],
                },
            },
        });

        team2 = await DbProvider.get().team.create({
            data: {
                id: testUsers[1].id + "team2",
                publicId: "team2-" + Math.random(),
                handle: "test-team-2",
                name: "Test Team 2",
                isPrivate: false,
                isOpenToAnyoneWithInvite: false,
                created_by: { connect: { id: testUsers[1].id } },
                members: {
                    create: [{
                        id: testUsers[1].id + "member2",
                        publicId: "member2-" + Math.random(),
                        user: { connect: { id: testUsers[1].id } },
                        permissions: JSON.stringify({ canEdit: true, canInvite: true }),
                    }],
                },
            },
        });

        publicTeam = await DbProvider.get().team.create({
            data: {
                id: testUsers[0].id + "publicteam",
                publicId: "publicteam-" + Math.random(),
                handle: "public-team",
                name: "Public Team",
                isPrivate: false,
                isOpenToAnyoneWithInvite: true,
                created_by: { connect: { id: testUsers[0].id } },
                members: {
                    create: [{
                        id: testUsers[0].id + "publicmember",
                        publicId: "publicmember-" + Math.random(),
                        user: { connect: { id: testUsers[0].id } },
                        permissions: JSON.stringify({ canEdit: true, canInvite: true }),
                    }],
                },
            },
        });

        // Seed member invites using database fixtures
        const team1Invites = await seedMemberInvites(DbProvider.get(), {
            teamId: team1.id,
            userIds: [testUsers[1].id], // User 1 is invited to team1
            withCustomMessages: true,
            messagePrefix: "You're invited to join Test Team 1",
        });
        invite1 = team1Invites[0];

        const team2Invites = await seedMemberInvites(DbProvider.get(), {
            teamId: team2.id,
            userIds: [testUsers[0].id], // User 0 is invited to team2
            withCustomMessages: true,
            messagePrefix: "You're invited to join Test Team 2",
        });
        invite2 = team2Invites[0];

        const publicTeamInvites = await seedMemberInvites(DbProvider.get(), {
            teamId: publicTeam.id,
            userIds: [testUsers[1].id], // User 1 is invited to publicTeam
            withCustomMessages: false,
        });
        publicInvite = publicTeamInvites[0];
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("invited user can view their invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id, // User 1 is invited to team1
                });

                const input: FindByIdInput = { id: invite1.id };
                const result = await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(invite1.id);
                expect(result.teamId).toBe(team1.id);
                expect(result.userId).toBe(testUsers[1].id);
                expect(result.status).toBe("Pending");
            });

            it("team admin can view invites", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created team1
                });

                const input: FindByIdInput = { id: invite1.id };
                const result = await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(invite1.id);
                expect(result.teamId).toBe(team1.id);
            });

            it("returns invite for public team with API key", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: FindByIdInput = { id: publicInvite.id };
                const result = await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(publicInvite.id);
            });
        });

        describe("invalid", () => {
            it("non-invited user cannot view invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not invited to team1
                });

                const input: FindByIdInput = { id: invite1.id };

                await expect(async () => {
                    await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot view invite", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = { id: invite1.id };

                await expect(async () => {
                    await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns invites visible to authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created team1 and is invited to team2
                });

                const input: MemberInviteSearchInput = { take: 10 };
                const result = await memberInvite.findMany({ input }, { req, res }, memberInvite_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThanOrEqual(1);

                const inviteIds = result.edges.map(e => e?.node?.id);
                // Should see invite2 (invited to team2) and possibly others
                expect(inviteIds).toContain(invite2.id);
            });

            it("filters invites by team", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created team1
                });

                const input: MemberInviteSearchInput = { teamId: team1.id, take: 10 };
                const result = await memberInvite.findMany({ input }, { req, res }, memberInvite_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only invite1
                expect(result.edges[0]?.node?.id).toBe(invite1.id);
                expect(result.edges[0]?.node?.teamId).toBe(team1.id);
            });

            it("filters invites by user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created team1
                });

                const input: MemberInviteSearchInput = {
                    teamId: team1.id,
                    userId: testUsers[1].id,
                    take: 10,
                };
                const result = await memberInvite.findMany({ input }, { req, res }, memberInvite_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only invite1
                expect(result.edges[0]?.node?.userId).toBe(testUsers[1].id);
            });
        });

        describe("invalid", () => {
            it("logged out user cannot view invites", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: MemberInviteSearchInput = { take: 10 };

                await expect(async () => {
                    await memberInvite.findMany({ input }, { req, res }, memberInvite_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("team admin can create invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created team1
                });

                // Use validation fixtures for API input
                const input: MemberInviteCreateInput = memberInviteTestDataFactory.createMinimal({
                    teamConnect: team1.id,
                    userConnect: testUsers[2].id, // Invite user 2
                    message: "Please join our team!",
                });

                const result = await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.teamId).toBe(team1.id);
                expect(result.userId).toBe(testUsers[2].id);
                expect(result.message).toBe("Please join our team!");
                expect(result.status).toBe("Pending");
            });

            it("API key with write permissions can create invite", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: MemberInviteCreateInput = memberInviteTestDataFactory.createMinimal({
                    teamConnect: team1.id,
                    userConnect: testUsers[2].id,
                });

                const result = await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.teamId).toBe(team1.id);
                expect(result.userId).toBe(testUsers[2].id);
            });
        });

        describe("invalid", () => {
            it("non-admin cannot create invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not admin of team1
                });

                const input: MemberInviteCreateInput = memberInviteTestDataFactory.createMinimal({
                    teamConnect: team1.id,
                    userConnect: testUsers[1].id,
                });

                await expect(async () => {
                    await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot create invite", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: MemberInviteCreateInput = memberInviteTestDataFactory.createMinimal({
                    teamConnect: team1.id,
                    userConnect: testUsers[2].id,
                });

                await expect(async () => {
                    await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("team admin can update invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created team1
                });

                const input: MemberInviteUpdateInput = {
                    id: invite1.id,
                    message: "Updated team invitation message",
                };

                const result = await memberInvite.updateOne({ input }, { req, res }, memberInvite_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(invite1.id);
                expect(result.message).toBe("Updated team invitation message");
            });
        });

        describe("invalid", () => {
            it("invited user cannot update invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id, // User 1 is invited but not admin
                });

                const input: MemberInviteUpdateInput = {
                    id: invite1.id,
                    message: "User trying to update",
                };

                await expect(async () => {
                    await memberInvite.updateOne({ input }, { req, res }, memberInvite_updateOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("acceptOne", () => {
        it("invited user can accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id, // User 1 is invited to team1
            });

            const input: FindByIdInput = { id: invite1.id };
            const result = await memberInvite.acceptOne({ input }, { req, res }, memberInvite_acceptOne);

            expect(result.status).toBe("Accepted");

            // Verify user is now a member
            const member = await DbProvider.get().member.findUnique({
                where: {
                    teamId_userId: {
                        teamId: team1.id,
                        userId: testUsers[1].id,
                    },
                },
            });
            expect(member).not.toBeNull();
        });

        it("non-invited user cannot accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[2].id, // User 2 is not invited
            });

            const input: FindByIdInput = { id: invite1.id };

            await expect(async () => {
                await memberInvite.acceptOne({ input }, { req, res }, memberInvite_acceptOne);
            }).rejects.toThrow();
        });

        it("cannot accept already accepted invite", async () => {
            // First, accept the invite
            const { req: req1, res: res1 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id,
            });
            await memberInvite.acceptOne({ input: { id: invite1.id } }, { req: req1, res: res1 }, memberInvite_acceptOne);

            // Then try to accept it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id,
            });

            await expect(async () => {
                await memberInvite.acceptOne({ input: { id: invite1.id } }, { req: req2, res: res2 }, memberInvite_acceptOne);
            }).rejects.toThrow();
        });
    });

    describe("declineOne", () => {
        it("invited user can decline invite", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id, // User 0 is invited to team2
            });

            const input: FindByIdInput = { id: invite2.id };
            const result = await memberInvite.declineOne({ input }, { req, res }, memberInvite_declineOne);

            expect(result.status).toBe("Declined");

            // Verify user is NOT a member
            const member = await DbProvider.get().member.findUnique({
                where: {
                    teamId_userId: {
                        teamId: team2.id,
                        userId: testUsers[0].id,
                    },
                },
            });
            expect(member).toBeNull();
        });

        it("team admin cannot decline invite (they should delete it)", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id, // User 1 created team2
            });

            const input: FindByIdInput = { id: invite2.id };

            await expect(async () => {
                await memberInvite.declineOne({ input }, { req, res }, memberInvite_declineOne);
            }).rejects.toThrow();
        });

        it("cannot decline already declined invite", async () => {
            // First, decline the invite
            const { req: req1, res: res1 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            await memberInvite.declineOne({ input: { id: invite2.id } }, { req: req1, res: res1 }, memberInvite_declineOne);

            // Then try to decline it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            await expect(async () => {
                await memberInvite.declineOne({ input: { id: invite2.id } }, { req: req2, res: res2 }, memberInvite_declineOne);
            }).rejects.toThrow();
        });
    });
});
