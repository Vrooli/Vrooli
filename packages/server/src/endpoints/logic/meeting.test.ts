import { FindByIdInput, MeetingCreateInput, MeetingSearchInput, MeetingUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions, seedMockAdminUser } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { meeting_createOne } from "../generated/meeting_createOne.js";
import { meeting_findMany } from "../generated/meeting_findMany.js";
import { meeting_findOne } from "../generated/meeting_findOne.js";
import { meeting_updateOne } from "../generated/meeting_updateOne.js";
import { meeting } from "./meeting.js";

// Test users and meeting IDs
let adminId: string;
const user1Id = uuid();
const user2Id = uuid();
const meeting1Id = uuid();
const meeting2Id = uuid();

describe("EndpointsMeeting", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let team1: any;
    let team2: any;

    before(() => {
        // Stub logger to suppress logs during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create two test users
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user1Id,
                name: "Test User 1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user2Id,
                name: "Test User 2",
            },
        });

        // Ensure admin user exists for update tests
        const admin = await seedMockAdminUser();
        adminId = admin.id.toString();
        // Create two teams for meeting ownership
        team1 = await DbProvider.get().team.create({ data: { permissions: "{}" } });
        team2 = await DbProvider.get().team.create({ data: { permissions: "{}" } });

        // Seed two meetings
        await DbProvider.get().meeting.create({ data: { id: meeting1Id, openToAnyoneWithInvite: true, showOnTeamProfile: false, team: { connect: { id: team1.id } } } });
        await DbProvider.get().meeting.create({ data: { id: meeting2Id, openToAnyoneWithInvite: false, showOnTeamProfile: true, team: { connect: { id: team2.id } } } });
    });

    after(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns meeting by id for any authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: meeting1Id };
                const result = await meeting.findOne({ input }, { req, res }, meeting_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(meeting1Id);
            });

            it("returns meeting by id when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: meeting2Id };
                const result = await meeting.findOne({ input }, { req, res }, meeting_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(meeting2Id);
            });

            it("returns meeting by id with API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: meeting1Id };
                const result = await meeting.findOne({ input }, { req, res }, meeting_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(meeting1Id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns meetings without filters for any authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingSearchInput = { take: 10 };
                const expectedIds = [
                    meeting1Id,
                    meeting2Id,
                ];
                const result = await meeting.findMany({ input }, { req, res }, meeting_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns meetings without filters for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingSearchInput = { take: 10 };
                const expectedIds = [
                    meeting1Id,
                    meeting2Id,
                ];
                const result = await meeting.findMany({ input }, { req, res }, meeting_findMany);
                expect(result).to.not.be.null;
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns meetings without filters for API key public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: MeetingSearchInput = { take: 10 };
                const expectedIds = [
                    meeting1Id,
                    meeting2Id,
                ];
                const result = await meeting.findMany({ input }, { req, res }, meeting_findMany);
                expect(result).to.not.be.null;
                assertFindManyResultIds(expect, result, expectedIds);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a meeting for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const newMeetingId = uuid();
                const input: MeetingCreateInput = { id: newMeetingId, teamConnect: team1.id };
                const result = await meeting.createOne({ input }, { req, res }, meeting_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newMeetingId);
                expect(result.team.id).to.equal(team1.id);
            });

            it("API key with write permissions can create meeting", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newMeetingId = uuid();
                const input: MeetingCreateInput = { id: newMeetingId, teamConnect: team1.id };
                const result = await meeting.createOne({ input }, { req, res }, meeting_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newMeetingId);
                expect(result.team.id).to.equal(team1.id);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create meeting", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingCreateInput = { id: uuid(), teamConnect: team1.id };
                try {
                    await meeting.createOne({ input }, { req, res }, meeting_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("allows admin to update a meeting", async () => {
                const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: MeetingUpdateInput = { id: meeting1Id, openToAnyoneWithInvite: false };
                const result = await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                expect(result).to.not.be.null;
                expect(result.openToAnyoneWithInvite).to.be.false;
            });
        });

        describe("invalid", () => {
            it("denies update for non-admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: MeetingUpdateInput = { id: meeting1Id, openToAnyoneWithInvite: false };
                try {
                    await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("denies update for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingUpdateInput = { id: meeting1Id, openToAnyoneWithInvite: false };
                try {
                    await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("throws when updating non-existent meeting as admin", async () => {
                const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: MeetingUpdateInput = { id: uuid(), openToAnyoneWithInvite: true };
                try {
                    await meeting.updateOne({ input }, { req, res }, meeting_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });
        });
    });
}); 
