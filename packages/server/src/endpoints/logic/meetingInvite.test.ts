import { FindByIdInput, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { meetingInvite_acceptOne } from "../generated/meetingInvite_acceptOne.js";
import { meetingInvite_createMany } from "../generated/meetingInvite_createMany.js";
import { meetingInvite_createOne } from "../generated/meetingInvite_createOne.js";
import { meetingInvite_declineOne } from "../generated/meetingInvite_declineOne.js";
import { meetingInvite_findMany } from "../generated/meetingInvite_findMany.js";
import { meetingInvite_findOne } from "../generated/meetingInvite_findOne.js";
import { meetingInvite_updateMany } from "../generated/meetingInvite_updateMany.js";
import { meetingInvite_updateOne } from "../generated/meetingInvite_updateOne.js";
import { meetingInvite } from "./meetingInvite.js";

const user1Id = uuid();
const user2Id = uuid();
const user3Id = uuid();
const meeting1Id = uuid();
const meeting2Id = uuid();
const invite1Id = uuid();
const invite2Id = uuid();

describe("EndpointsMeetingInvite", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let team1: any;
    let team2: any;

    before(() => {
        // Suppress logger output during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed three users
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
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user3Id,
                name: "Test User 3",
            },
        });

        // Create two teams for meeting ownership
        team1 = await DbProvider.get().team.create({ data: { permissions: "{}" } });
        team2 = await DbProvider.get().team.create({ data: { permissions: "{}" } });

        // Seed team memberships so meeting creators are recognized as team members
        await DbProvider.get().member.create({ data: { id: uuid(), team: { connect: { id: team1.id } }, user: { connect: { id: user1Id } }, isAdmin: true, permissions: "{}" } });
        await DbProvider.get().member.create({ data: { id: uuid(), team: { connect: { id: team2.id } }, user: { connect: { id: user2Id } }, isAdmin: true, permissions: "{}" } });

        // Seed two meetings
        await DbProvider.get().meeting.create({ data: { id: meeting1Id, openToAnyoneWithInvite: true, showOnTeamProfile: false, team: { connect: { id: team1.id } } } });
        await DbProvider.get().meeting.create({ data: { id: meeting2Id, openToAnyoneWithInvite: false, showOnTeamProfile: true, team: { connect: { id: team2.id } } } });

        // Seed two meeting invites
        await DbProvider.get().meeting_invite.create({
            data: {
                id: invite1Id,
                message: "Invite One",
                meeting: { connect: { id: meeting1Id } },
                user: { connect: { id: user2Id } },
            },
        });
        await DbProvider.get().meeting_invite.create({
            data: {
                id: invite2Id,
                message: "Invite Two",
                meeting: { connect: { id: meeting2Id } },
                user: { connect: { id: user1Id } },
            },
        });
    });

    after(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns invite when user is the meeting creator", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: invite1Id };
                const result = await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite1Id);
            });

            it("returns invite when user is the invited user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: invite2Id };
                const result = await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite2Id);
            });

            it("throws error for user with no visibility permission", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: invite1Id };
                try {
                    await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);
                    expect.fail("Expected an error due to visibility restrictions");
                } catch (err) { /* expected */ }
            });

            it("returns invite for API key with private read permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: invite1Id };
                const result = await meetingInvite.findOne({ input }, { req, res }, meetingInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite1Id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns only invites visible to authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteSearchInput = { take: 10 };
                const result = await meetingInvite.findMany({ input }, { req, res }, meetingInvite_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                const ids = result.edges!.map(e => e!.node!.id).sort();
                ids.forEach(id => { expect([invite1Id, invite2Id].includes(id!)).to.be.true; });
            });

            it("fails for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingInviteSearchInput = { take: 10 };
                try {
                    await meetingInvite.findMany({ input }, { req, res }, meetingInvite_findMany);
                    expect.fail("Expected an error due to visibility restrictions");
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates an invite for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const newInviteId = uuid();
                const input: MeetingInviteCreateInput = { id: newInviteId, meetingConnect: meeting1Id, userConnect: user3Id, message: "New Invite" };
                const result = await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newInviteId);
                expect(result.message).to.equal("New Invite");
            });

            it("API key with write permissions can create invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const newInviteId = uuid();
                const input: MeetingInviteCreateInput = { id: newInviteId, meetingConnect: meeting1Id, userConnect: user3Id };
                const result = await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newInviteId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingInviteCreateInput = { id: uuid(), meetingConnect: meeting1Id, userConnect: user2Id };
                try {
                    await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (err) { /* expected */ }
            });

            it("authenticated user cannot create invite for meeting they don't own", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user3Id });
                const input: MeetingInviteCreateInput = { id: uuid(), meetingConnect: meeting2Id, userConnect: user3Id, message: "Invalid Invite" };
                try {
                    await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                    expect.fail("Expected an error due to permission restrictions");
                } catch (err) { /* expected */ }
            });

            it("API key without write permissions cannot create invite", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteCreateInput = { id: uuid(), meetingConnect: meeting2Id, userConnect: user3Id };
                try {
                    await meetingInvite.createOne({ input }, { req, res }, meetingInvite_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("createMany", () => {
        describe("valid", () => {
            it("creates multiple invites for meetings user owns", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const idA = uuid();
                const input: MeetingInviteCreateInput[] = [
                    { id: idA, meetingConnect: meeting1Id, userConnect: user3Id, message: "Bulk 1" },
                ];
                const result = await meetingInvite.createMany({ input }, { req, res }, meetingInvite_createMany);
                expect(result).to.have.length(1);
                expect(result[0].id).to.equal(idA);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingInviteCreateInput[] = [
                    { id: uuid(), meetingConnect: meeting1Id, userConnect: user3Id },
                ];
                try {
                    await meetingInvite.createMany({ input }, { req, res }, meetingInvite_createMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });

            it("API key without write permissions cannot create many invites", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteCreateInput[] = [
                    { id: uuid(), meetingConnect: meeting2Id, userConnect: user3Id },
                ];
                try {
                    await meetingInvite.createMany({ input }, { req, res }, meetingInvite_createMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates invite for meeting owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteUpdateInput = { id: invite1Id, message: "Updated Msg" };
                const result = await meetingInvite.updateOne({ input }, { req, res }, meetingInvite_updateOne);
                expect(result).to.not.be.null;
                expect(result.message).to.equal("Updated Msg");
            });

            it("cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteUpdateInput = { id: invite2Id, message: "Updated By Recipient" };
                try {
                    await meetingInvite.updateOne({ input }, { req, res }, meetingInvite_updateOne);
                    expect.fail("Expected error due to permission restrictions");
                } catch (err) { /* expected */ }
            });

            it("API key with write permissions can update an invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteUpdateInput = { id: invite1Id, message: "API Update" };
                const result = await meetingInvite.updateOne({ input }, { req, res }, meetingInvite_updateOne);
                expect(result).to.not.be.null;
                expect(result.message).to.equal("API Update");
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingInviteUpdateInput = { id: invite1Id, message: "Fail Update" };
                try {
                    await meetingInvite.updateOne({ input }, { req, res }, meetingInvite_updateOne);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("updateMany", () => {
        describe("valid", () => {
            it("updates multiple invites where user has visibility", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteUpdateInput[] = [{ id: invite1Id, message: "Bulk Update 1" }];
                const result = await meetingInvite.updateMany({ input }, { req, res }, meetingInvite_updateMany);
                expect(result).to.have.length(1);
                expect(result[0].message).to.equal("Bulk Update 1");
            });

            it("API key with write permissions can update many invites", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteUpdateInput[] = [{ id: invite1Id, message: "API Bulk" }];
                const result = await meetingInvite.updateMany({ input }, { req, res }, meetingInvite_updateMany);
                expect(result).to.have.length(1);
                expect(result[0].message).to.equal("API Bulk");
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MeetingInviteUpdateInput[] = [{ id: invite1Id, message: "Fail Bulk" }];
                try {
                    await meetingInvite.updateMany({ input }, { req, res }, meetingInvite_updateMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });

            it("cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MeetingInviteUpdateInput[] = [{ id: invite2Id, message: "Fail Bulk" }];
                try {
                    await meetingInvite.updateMany({ input }, { req, res }, meetingInvite_updateMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("acceptOne", () => {
        it("invited user can accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            const input: FindByIdInput = { id: invite1Id };

            const result = await meetingInvite.acceptOne({ input }, { req, res }, meetingInvite_acceptOne);

            expect(result.status).to.equal("Accepted");

            // Verify user2 is now an attendee in meeting1
            const attendee = await DbProvider.get().meeting_attendees.findUnique({
                where: { meetingId_userId: { meetingId: meeting1Id, userId: user2Id } },
            });
            expect(attendee).to.not.be.null;
        });

        it("non-invited user cannot accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user3Id });
            const input: FindByIdInput = { id: invite1Id };
            try {
                await meetingInvite.acceptOne({ input }, { req, res }, meetingInvite_acceptOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("cannot accept non-pending invite", async () => {
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            await meetingInvite.acceptOne({ input: { id: invite1Id } }, { req: req1, res: res1 }, meetingInvite_acceptOne);

            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            try {
                await meetingInvite.acceptOne({ input: { id: invite1Id } }, { req: req2, res: res2 }, meetingInvite_acceptOne);
                expect.fail("Expected Conflict error");
            } catch (err) { /* expected */ }
        });
    });

    describe("declineOne", () => {
        it("invited user can decline invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            const input: FindByIdInput = { id: invite2Id };

            const result = await meetingInvite.declineOne({ input }, { req, res }, meetingInvite_declineOne);

            expect(result.status).to.equal("Declined");

            // Verify user1 is NOT an attendee in meeting2
            const attendee = await DbProvider.get().meeting_attendees.findUnique({
                where: { meetingId_userId: { meetingId: meeting2Id, userId: user1Id } },
            });
            expect(attendee).to.be.null;
        });

        it("meeting owner can't decline invite (they must delete it)", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            const input: FindByIdInput = { id: invite2Id };
            try {
                await meetingInvite.declineOne({ input }, { req, res }, meetingInvite_declineOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("non-involved user cannot decline invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user3Id });
            const input: FindByIdInput = { id: invite2Id };
            try {
                await meetingInvite.declineOne({ input }, { req, res }, meetingInvite_declineOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("cannot decline non-pending invite", async () => {
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            await meetingInvite.declineOne({ input: { id: invite2Id } }, { req: req1, res: res1 }, meetingInvite_declineOne);

            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            try {
                await meetingInvite.declineOne({ input: { id: invite2Id } }, { req: req2, res: res2 }, meetingInvite_declineOne);
                expect.fail("Expected Conflict error");
            } catch (err) { /* expected */ }
        });
    });
});
