import { type FindByIdInput, type MeetingInviteCreateInput, type MeetingInviteSearchInput, type MeetingInviteUpdateInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { meetingInvite_acceptOne } from "../generated/meetingInvite_acceptOne.js";
import { meetingInvite_createOne } from "../generated/meetingInvite_createOne.js";
import { meetingInvite_declineOne } from "../generated/meetingInvite_declineOne.js";
import { meetingInvite_findMany } from "../generated/meetingInvite_findMany.js";
import { meetingInvite_findOne } from "../generated/meetingInvite_findOne.js";
import { meetingInvite_updateOne } from "../generated/meetingInvite_updateOne.js";
import { meetingInvite } from "./meetingInvite.js";

// Import database fixtures for seeding
import { MeetingDbFactory } from "../../__test/fixtures/db/meetingFixtures.js";
import { seedMeetingInvites } from "../../__test/fixtures/db/meetingInviteFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing
import { meetingInviteTestDataFactory } from "@vrooli/shared";

describe("EndpointsMeetingInvite", () => {
    let testUsers: any[];
    let team1: any;
    let team2: any;
    let meeting1: any;
    let meeting2: any;
    let publicMeeting: any;
    let invite1: any;
    let invite2: any;
    let publicInvite: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        try {
            const prisma = DbProvider.get();
            if (prisma) {
                testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true
            }
        } catch (error) {
            // If database is not initialized, skip cleanup
        }
    });

        // Create teams for meetings
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

        // Create meetings using database fixtures
        meeting1 = await DbProvider.get().meeting.create({
            data: MeetingDbFactory.createMinimal({
                name: "Test Meeting 1",
                description: "First test meeting",
                openToAnyoneWithInvite: false,
                restrictedToRoles: false,
                team: { connect: { id: team1.id } },
                created_by: { connect: { id: testUsers[0].id } },
            }),
        });

        meeting2 = await DbProvider.get().meeting.create({
            data: MeetingDbFactory.createMinimal({
                name: "Test Meeting 2",
                description: "Second test meeting",
                openToAnyoneWithInvite: false,
                restrictedToRoles: false,
                team: { connect: { id: team2.id } },
                created_by: { connect: { id: testUsers[1].id } },
            }),
        });

        publicMeeting = await DbProvider.get().meeting.create({
            data: MeetingDbFactory.createMinimal({
                name: "Public Meeting",
                description: "A public meeting",
                openToAnyoneWithInvite: true,
                restrictedToRoles: false,
                team: { connect: { id: team1.id } },
                created_by: { connect: { id: testUsers[0].id } },
            }),
        });

        // Seed meeting invites using database fixtures
        const meeting1Invites = await seedMeetingInvites(DbProvider.get(), {
            meetingId: meeting1.id,
            userIds: [testUsers[1].id], // User 1 is invited to meeting1
            withCustomMessages: true,
            messagePrefix: "You're invited to Test Meeting 1",
        });
        invite1 = meeting1Invites[0];

        const meeting2Invites = await seedMeetingInvites(DbProvider.get(), {
            meetingId: meeting2.id,
            userIds: [testUsers[0].id], // User 0 is invited to meeting2
            withCustomMessages: true,
            messagePrefix: "You're invited to Test Meeting 2",
        });
        invite2 = meeting2Invites[0];

        const publicMeetingInvites = await seedMeetingInvites(DbProvider.get(), {
            meetingId: publicMeeting.id,
            userIds: [testUsers[1].id], // User 1 is invited to publicMeeting
            withCustomMessages: false,
        });
        publicInvite = publicMeetingInvites[0];
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
                    id: testUsers[1].id, // User 1 is invited to meeting1
                });

                const input: FindByIdInput = { id: invite1.id };
                const result = await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(invite1.id);
                expect(result.meetingId).toBe(meeting1.id);
                expect(result.userId).toBe(testUsers[1].id);
                expect(result.status).toBe("Pending");
            });

            it("meeting creator can view invites", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created meeting1
                });

                const input: FindByIdInput = { id: invite1.id };
                const result = await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(invite1.id);
                expect(result.meetingId).toBe(meeting1.id);
            });

            it("returns invite for public meeting with API key", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: FindByIdInput = { id: publicInvite.id };
                const result = await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(publicInvite.id);
            });
        });

        describe("invalid", () => {
            it("non-invited user cannot view invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not invited to meeting1
                });

                const input: FindByIdInput = { id: invite1.id };

                await expect(async () => {
                    await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot view invite", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = { id: invite1.id };

                await expect(async () => {
                    await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns invites visible to authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created meeting1 and is invited to meeting2
                });

                const input: MeetingInviteSearchInput = { take: 10 };
                const result = await meetingInvite.findMany({ input }, { req, res }, meetingInvite_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThanOrEqual(1);

                const inviteIds = result.edges.map(e => e?.node?.id);
                // Should see invite2 (invited to meeting2) and possibly others
                expect(inviteIds).toContain(invite2.id);
            });

            it("filters invites by meeting", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created meeting1
                });

                const input: MeetingInviteSearchInput = { meetingId: meeting1.id, take: 10 };
                const result = await meetingInvite.findMany({ input }, { req, res }, meetingInvite_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only invite1
                expect(result.edges[0]?.node?.id).toBe(invite1.id);
                expect(result.edges[0]?.node?.meetingId).toBe(meeting1.id);
            });

            it("filters invites by user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created meeting1
                });

                const input: MeetingInviteSearchInput = {
                    meetingId: meeting1.id,
                    userId: testUsers[1].id,
                    take: 10
                };
                const result = await meetingInvite.findMany({ input }, { req, res }, meetingInvite_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only invite1
                expect(result.edges[0]?.node?.userId).toBe(testUsers[1].id);
            });
        });

        describe("invalid", () => {
            it("logged out user cannot view invites", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: MeetingInviteSearchInput = { take: 10 };

                await expect(async () => {
                    await meetingInvite.findMany({ input }, { req, res }, meetingInvite_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("meeting creator can create invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created meeting1
                });

                // Use validation fixtures for API input
                const input: MeetingInviteCreateInput = meetingInviteTestDataFactory.createMinimal({
                    meetingConnect: meeting1.id,
                    userConnect: testUsers[2].id, // Invite user 2
                    message: "Please join our meeting!",
                });

                const result = await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.meetingId).toBe(meeting1.id);
                expect(result.userId).toBe(testUsers[2].id);
                expect(result.message).toBe("Please join our meeting!");
                expect(result.status).toBe("Pending");
            });

            it("API key with write permissions can create invite", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: MeetingInviteCreateInput = meetingInviteTestDataFactory.createMinimal({
                    meetingConnect: meeting1.id,
                    userConnect: testUsers[2].id,
                });

                const result = await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.meetingId).toBe(meeting1.id);
                expect(result.userId).toBe(testUsers[2].id);
            });
        });

        describe("invalid", () => {
            it("non-creator cannot create invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 didn't create meeting1
                });

                const input: MeetingInviteCreateInput = meetingInviteTestDataFactory.createMinimal({
                    meetingConnect: meeting1.id,
                    userConnect: testUsers[1].id,
                });

                await expect(async () => {
                    await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot create invite", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: MeetingInviteCreateInput = meetingInviteTestDataFactory.createMinimal({
                    meetingConnect: meeting1.id,
                    userConnect: testUsers[2].id,
                });

                await expect(async () => {
                    await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("meeting creator can update invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 created meeting1
                });

                const input: MeetingInviteUpdateInput = {
                    id: invite1.id,
                    message: "Updated meeting invitation message",
                };

                const result = await meetingInvite.updateOne({ input }, { req, res }, meetingInvite_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(invite1.id);
                expect(result.message).toBe("Updated meeting invitation message");
            });
        });

        describe("invalid", () => {
            it("invited user cannot update invite", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id, // User 1 is invited but not creator
                });

                const input: MeetingInviteUpdateInput = {
                    id: invite1.id,
                    message: "User trying to update",
                };

                await expect(async () => {
                    await meetingInvite.updateOne({ input }, { req, res }, meetingInvite_updateOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("acceptOne", () => {
        it("invited user can accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id, // User 1 is invited to meeting1
            });

            const input: FindByIdInput = { id: invite1.id };
            const result = await meetingInvite.acceptOne({ input }, { req, res }, meetingInvite_acceptOne);

            expect(result.status).toBe("Accepted");

            // Verify user is now an attendee
            const attendee = await DbProvider.get().meeting_attendees.findUnique({
                where: {
                    meetingId_userId: {
                        meetingId: meeting1.id,
                        userId: testUsers[1].id
                    }
                },
            });
            expect(attendee).not.toBeNull();
        });

        it("non-invited user cannot accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[2].id, // User 2 is not invited
            });

            const input: FindByIdInput = { id: invite1.id };

            await expect(async () => {
                await meetingInvite.acceptOne({ input }, { req, res }, meetingInvite_acceptOne);
            }).rejects.toThrow();
        });

        it("cannot accept already accepted invite", async () => {
            // First, accept the invite
            const { req: req1, res: res1 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id,
            });
            await meetingInvite.acceptOne({ input: { id: invite1.id } }, { req: req1, res: res1 }, meetingInvite_acceptOne);

            // Then try to accept it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id,
            });

            await expect(async () => {
                await meetingInvite.acceptOne({ input: { id: invite1.id } }, { req: req2, res: res2 }, meetingInvite_acceptOne);
            }).rejects.toThrow();
        });
    });

    describe("declineOne", () => {
        it("invited user can decline invite", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id, // User 0 is invited to meeting2
            });

            const input: FindByIdInput = { id: invite2.id };
            const result = await meetingInvite.declineOne({ input }, { req, res }, meetingInvite_declineOne);

            expect(result.status).toBe("Declined");

            // Verify user is NOT an attendee
            const attendee = await DbProvider.get().meeting_attendees.findUnique({
                where: {
                    meetingId_userId: {
                        meetingId: meeting2.id,
                        userId: testUsers[0].id
                    }
                },
            });
            expect(attendee).toBeNull();
        });

        it("meeting creator cannot decline invite (they should delete it)", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id, // User 1 created meeting2
            });

            const input: FindByIdInput = { id: invite2.id };

            await expect(async () => {
                await meetingInvite.declineOne({ input }, { req, res }, meetingInvite_declineOne);
            }).rejects.toThrow();
        });

        it("cannot decline already declined invite", async () => {
            // First, decline the invite
            const { req: req1, res: res1 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            await meetingInvite.declineOne({ input: { id: invite2.id } }, { req: req1, res: res1 }, meetingInvite_declineOne);

            // Then try to decline it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            await expect(async () => {
                await meetingInvite.declineOne({ input: { id: invite2.id } }, { req: req2, res: res2 }, meetingInvite_declineOne);
            }).rejects.toThrow();
        });
    });
});