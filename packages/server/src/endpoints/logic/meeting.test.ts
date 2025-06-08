import { type FindByIdInput, type MeetingCreateInput, type MeetingSearchInput, type MeetingUpdateInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions, seedMockAdminUser } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { meeting_createOne } from "../generated/meeting_createOne.js";
import { meeting_findMany } from "../generated/meeting_findMany.js";
import { meeting_findOne } from "../generated/meeting_findOne.js";
import { meeting_updateOne } from "../generated/meeting_updateOne.js";
import { meeting } from "./meeting.js";

// Import database fixtures for seeding
import { seedMeetings } from "../../__test/fixtures/db/meetingFixtures.js";
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing
import { meetingTestDataFactory } from "@vrooli/shared/validation/models";

describe("EndpointsMeeting", () => {
    let testUsers: any[];
    let adminUser: any;
    let teams: any[];
    let meetings: any[];

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
        adminUser = await seedMockAdminUser();

        // Create two teams for meeting ownership
        teams = await Promise.all([
            DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    permissions: "{}",
                    owner: { connect: { id: testUsers[0].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Team 1",
                        },
                    },
                },
            }),
            DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: UserDbFactory.createMinimal().publicId,
                    permissions: "{}",
                    owner: { connect: { id: testUsers[1].id } },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: "Team 2",
                        },
                    },
                },
            }),
        ]);

        // Seed meetings using database fixtures
        meetings = [
            ...(await seedMeetings(DbProvider.get(), {
                teamId: teams[0].id,
                createdById: testUsers[0].id,
                count: 1,
                withInvites: [{ userId: testUsers[1].id }],
            })),
            ...(await seedMeetings(DbProvider.get(), {
                teamId: teams[1].id,
                createdById: testUsers[1].id,
                count: 1,
            })),
        ];
    });

    afterAll(async () => {
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns meeting by id for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: FindByIdInput = { id: meetings[0].id };
                const result = await meeting.findOne({ input }, { req, res }, meeting_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(meetings[0].id);
            });

            it("returns meeting by id when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: meetings[1].id };
                const result = await meeting.findOne({ input }, { req, res }, meeting_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(meetings[1].id);
            });

            it("returns meeting by id with API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: FindByIdInput = { id: meetings[0].id };
                const result = await meeting.findOne({ input }, { req, res }, meeting_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(meetings[0].id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns meetings without filters for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: MeetingSearchInput = { take: 10 };
                const expectedIds = meetings.map(m => m.id);
                const result = await meeting.findMany({ input }, { req, res }, meeting_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns meetings without filters for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingSearchInput = { take: 10 };
                const expectedIds = meetings.map(m => m.id);
                const result = await meeting.findMany({ input }, { req, res }, meeting_findMany);
                expect(result).not.toBeNull();
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns meetings without filters for API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });
                const input: MeetingSearchInput = { take: 10 };
                const expectedIds = meetings.map(m => m.id);
                const result = await meeting.findMany({ input }, { req, res }, meeting_findMany);
                expect(result).not.toBeNull();
                assertFindManyResultIds(expect, result, expectedIds);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a meeting for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                // Use validation fixtures for API input
                const input: MeetingCreateInput = meetingTestDataFactory.createMinimal({
                    teamConnect: teams[0].id,
                    openToAnyoneWithInvite: true,
                    translationsCreate: [{
                        language: "en",
                        name: "New Meeting",
                        description: "A test meeting",
                    }],
                });

                const result = await meeting.createOne({ input }, { req, res }, meeting_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
                expect(result.openToAnyoneWithInvite).toBe(true);
            });

            it("API key with write permissions can create meeting", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                // Use complete fixture for comprehensive test
                const input: MeetingCreateInput = meetingTestDataFactory.createComplete({
                    teamConnect: teams[0].id,
                });

                const result = await meeting.createOne({ input }, { req, res }, meeting_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("throws error for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: MeetingCreateInput = meetingTestDataFactory.createMinimal({
                    teamConnect: teams[0].id,
                });

                await expect(async () => {
                    await meeting.createOne({ input }, { req, res }, meeting_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates meeting for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id
                });

                const input: MeetingUpdateInput = {
                    id: meetings[0].id,
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                };

                const result = await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                expect(result).not.toBeNull();
                expect(result.openToAnyoneWithInvite).toBe(false);
                expect(result.showOnTeamProfile).toBe(true);
            });

            it("admin can update any meeting", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: adminUser.id
                });

                const input: MeetingUpdateInput = {
                    id: meetings[1].id,
                    translationsUpdate: [{
                        id: meetings[1].translations[0].id,
                        name: "Admin Updated Meeting",
                    }],
                };

                const result = await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                expect(result).not.toBeNull();
                expect(result.translations[0].name).toBe("Admin Updated Meeting");
            });
        });

        describe("invalid", () => {
            it("throws error for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: MeetingUpdateInput = {
                    id: meetings[0].id,
                    showOnTeamProfile: false,
                };

                await expect(async () => {
                    await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                }).rejects.toThrow();
            });
        });
    });
});