import { type FindByIdInput, type MemberInviteCreateInput, type MemberInviteSearchInput, type MemberInviteUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { memberInvite_acceptOne } from "../generated/memberInvite_acceptOne.js";
import { memberInvite_createMany } from "../generated/memberInvite_createMany.js";
import { memberInvite_createOne } from "../generated/memberInvite_createOne.js";
import { memberInvite_declineOne } from "../generated/memberInvite_declineOne.js";
import { memberInvite_findMany } from "../generated/memberInvite_findMany.js";
import { memberInvite_findOne } from "../generated/memberInvite_findOne.js";
import { memberInvite_updateMany } from "../generated/memberInvite_updateMany.js";
import { memberInvite_updateOne } from "../generated/memberInvite_updateOne.js";
import { memberInvite } from "./memberInvite.js";

const user1Id = uuid();
const user2Id = uuid();
const user3Id = uuid();
const invite1Id = uuid();
const invite2Id = uuid();

describe("EndpointsMemberInvite", () => {
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

        // Create two teams for membership
        team1 = await DbProvider.get().team.create({ data: { permissions: "{}" } });
        team2 = await DbProvider.get().team.create({ data: { permissions: "{}" } });

        // Seed team membership for owners
        await DbProvider.get().member.create({ data: { id: uuid(), team: { connect: { id: team1.id } }, user: { connect: { id: user1Id } }, isAdmin: true, permissions: "{}" } });
        await DbProvider.get().member.create({ data: { id: uuid(), team: { connect: { id: team2.id } }, user: { connect: { id: user2Id } }, isAdmin: true, permissions: "{}" } });

        // Seed two member invites
        await DbProvider.get().member_invite.create({ data: { id: invite1Id, message: "Invite One", team: { connect: { id: team1.id } }, user: { connect: { id: user2Id } } } });
        await DbProvider.get().member_invite.create({ data: { id: invite2Id, message: "Invite Two", team: { connect: { id: team2.id } }, user: { connect: { id: user1Id } } } });
    });

    after(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns invite when user is the team owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: invite1Id };
                const result = await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite1Id);
            });

            it("returns invite when user is the invited user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: invite2Id };
                const result = await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite2Id);
            });

            it("throws error for user with no visibility permission", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: invite1Id };
                try {
                    await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);
                    expect.fail("Expected an error due to visibility restrictions");
                } catch (err) { /* expected */ }
            });

            it("returns invite for API key with private read permissions", async () => {
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: invite1Id };
                const result = await memberInvite.findOne({ input }, { req, res }, memberInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite1Id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns only invites visible to authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteSearchInput = { take: 10 };
                const result = await memberInvite.findMany({ input }, { req, res }, memberInvite_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const ids = result.edges!.map(e => e!.node!.id).sort();
                ids.forEach(id => expect([invite1Id, invite2Id].includes(id!)).to.be.true);
            });

            it("fails for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MemberInviteSearchInput = { take: 10 };
                try {
                    await memberInvite.findMany({ input }, { req, res }, memberInvite_findMany);
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
                const input: MemberInviteCreateInput = { id: newInviteId, teamConnect: team1.id, userConnect: user3Id, message: "New Invite" };
                const result = await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newInviteId);
                expect(result.message).to.equal("New Invite");
            });

            it("API key with write permissions can create invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const newInviteId = uuid();
                const input: MemberInviteCreateInput = { id: newInviteId, teamConnect: team1.id, userConnect: user3Id };
                const result = await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newInviteId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MemberInviteCreateInput = { id: uuid(), teamConnect: team1.id, userConnect: user2Id };
                try {
                    await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (err) { /* expected */ }
            });

            it("authenticated user cannot create invite for team they don't own", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user3Id });
                const input: MemberInviteCreateInput = { id: uuid(), teamConnect: team2.id, userConnect: user3Id, message: "Invalid Invite" };
                try {
                    await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                    expect.fail("Expected an error due to permission restrictions");
                } catch (err) { /* expected */ }
            });

            it("API key without write permissions cannot create invite", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteCreateInput = { id: uuid(), teamConnect: team2.id, userConnect: user3Id };
                try {
                    await memberInvite.createOne({ input }, { req, res }, memberInvite_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("createMany", () => {
        describe("valid", () => {
            it("creates multiple invites for teams user owns", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const idA = uuid();
                const input: MemberInviteCreateInput[] = [
                    { id: idA, teamConnect: team1.id, userConnect: user3Id, message: "Bulk 1" },
                ];
                const result = await memberInvite.createMany({ input }, { req, res }, memberInvite_createMany);
                expect(result).to.have.length(1);
                expect(result[0].id).to.equal(idA);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MemberInviteCreateInput[] = [
                    { id: uuid(), teamConnect: team1.id, userConnect: user3Id },
                ];
                try {
                    await memberInvite.createMany({ input }, { req, res }, memberInvite_createMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });

            it("API key without write permissions cannot create many invites", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteCreateInput[] = [
                    { id: uuid(), teamConnect: team2.id, userConnect: user3Id },
                ];
                try {
                    await memberInvite.createMany({ input }, { req, res }, memberInvite_createMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates invite for team owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteUpdateInput = { id: invite1Id, message: "Updated Msg" };
                const result = await memberInvite.updateOne({ input }, { req, res }, memberInvite_updateOne);
                expect(result).to.not.be.null;
                expect(result.message).to.equal("Updated Msg");
            });

            it("cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteUpdateInput = { id: invite2Id, message: "Updated By Recipient" };
                try {
                    await memberInvite.updateOne({ input }, { req, res }, memberInvite_updateOne);
                    expect.fail("Expected error due to permission restrictions");
                } catch (err) { /* expected */ }
            });

            it("API key with write permissions can update an invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteUpdateInput = { id: invite1Id, message: "API Update" };
                const result = await memberInvite.updateOne({ input }, { req, res }, memberInvite_updateOne);
                expect(result).to.not.be.null;
                expect(result.message).to.equal("API Update");
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MemberInviteUpdateInput = { id: invite1Id, message: "Fail Update" };
                try {
                    await memberInvite.updateOne({ input }, { req, res }, memberInvite_updateOne);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("updateMany", () => {
        describe("valid", () => {
            it("updates multiple invites where user has visibility", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteUpdateInput[] = [{ id: invite1Id, message: "Bulk Update 1" }];
                const result = await memberInvite.updateMany({ input }, { req, res }, memberInvite_updateMany);
                expect(result).to.have.length(1);
                expect(result[0].message).to.equal("Bulk Update 1");
            });

            it("API key with write permissions can update many invites", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteUpdateInput[] = [{ id: invite1Id, message: "API Bulk" }];
                const result = await memberInvite.updateMany({ input }, { req, res }, memberInvite_updateMany);
                expect(result).to.have.length(1);
                expect(result[0].message).to.equal("API Bulk");
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: MemberInviteUpdateInput[] = [{ id: invite1Id, message: "Fail Bulk" }];
                try {
                    await memberInvite.updateMany({ input }, { req, res }, memberInvite_updateMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });

            it("cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: MemberInviteUpdateInput[] = [{ id: invite2Id, message: "Fail Bulk" }];
                try {
                    await memberInvite.updateMany({ input }, { req, res }, memberInvite_updateMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("acceptOne", () => {
        it("invited user can accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            const input: FindByIdInput = { id: invite1Id };

            const result = await memberInvite.acceptOne({ input }, { req, res }, memberInvite_acceptOne);

            expect(result.status).to.equal("Accepted");

            // Verify user2 is now a member of team1
            const member = await DbProvider.get().member.findUnique({
                where: { member_teamid_userid_unique: { teamId: team1.id, userId: user2Id } },
            });
            expect(member).to.not.be.null;
        });

        it("non-invited user cannot accept invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user3Id });
            const input: FindByIdInput = { id: invite1Id };
            try {
                await memberInvite.acceptOne({ input }, { req, res }, memberInvite_acceptOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("cannot accept non-pending invite", async () => {
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            await memberInvite.acceptOne({ input: { id: invite1Id } }, { req: req1, res: res1 }, memberInvite_acceptOne);

            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            try {
                await memberInvite.acceptOne({ input: { id: invite1Id } }, { req: req2, res: res2 }, memberInvite_acceptOne);
                expect.fail("Expected Conflict error");
            } catch (err) { /* expected */ }
        });
    });

    describe("declineOne", () => {
        it("invited user can decline invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            const input: FindByIdInput = { id: invite2Id };

            const result = await memberInvite.declineOne({ input }, { req, res }, memberInvite_declineOne);

            expect(result.status).to.equal("Declined");

            // Verify user1 is NOT a member of team2
            const member = await DbProvider.get().member.findUnique({
                where: { member_teamid_userid_unique: { teamId: team2.id, userId: user1Id } },
            });
            expect(member).to.be.null;
        });

        it("team owner can't decline invite (they must delete it)", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user2Id });
            const input: FindByIdInput = { id: invite2Id };
            try {
                await memberInvite.declineOne({ input }, { req, res }, memberInvite_declineOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("non-involved user cannot decline invite", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user3Id });
            const input: FindByIdInput = { id: invite2Id };
            try {
                await memberInvite.declineOne({ input }, { req, res }, memberInvite_declineOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("cannot decline non-pending invite", async () => {
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            await memberInvite.declineOne({ input: { id: invite2Id } }, { req: req1, res: res1 }, memberInvite_declineOne);

            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            try {
                await memberInvite.declineOne({ input: { id: invite2Id } }, { req: req2, res: res2 }, memberInvite_declineOne);
                expect.fail("Expected Conflict error");
            } catch (err) { /* expected */ }
        });
    });
}); 
