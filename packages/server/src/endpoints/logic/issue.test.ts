import { ApiKeyPermission, FindByIdInput, IssueCloseInput, IssueCreateInput, IssueSearchInput, IssueStatus, IssueUpdateInput, SEEDED_IDS, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { issue_closeOne } from "../generated/issue_closeOne.js";
import { issue_createOne } from "../generated/issue_createOne.js";
import { issue_findMany } from "../generated/issue_findMany.js";
import { issue_findOne } from "../generated/issue_findOne.js";
import { issue_updateOne } from "../generated/issue_updateOne.js";
import { issue } from "./issue.js";

// Generate test user IDs
const user1Id = uuid();
const user2Id = uuid();
let team1: any;
let issueUser1: any;
let issueUser2: any;
let label1Id: string;

/**
 * Test suite for the Issue endpoint (findOne, findMany, createOne, updateOne, closeOne)
 */
describe("EndpointsIssue", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Stub logger to suppress output
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().session.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().team.deleteMany();
        await DbProvider.get().issue_translation.deleteMany();
        await DbProvider.get().issue.deleteMany();
        await DbProvider.get().label_translation.deleteMany();
        await DbProvider.get().label.deleteMany();

        // Create a team for issue ownership
        team1 = await DbProvider.get().team.create({ data: { permissions: "{}" } });

        // Create two users
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "Test User 1",
                handle: "test-user-1",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "Test User 2",
                handle: "test-user-2",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });

        // Seed two issues: one for each user
        issueUser1 = await DbProvider.get().issue.create({
            data: {
                id: uuid(),
                createdBy: { connect: { id: user1Id } },
                team: { connect: { id: team1.id } },
            },
        });
        issueUser2 = await DbProvider.get().issue.create({
            data: {
                id: uuid(),
                createdBy: { connect: { id: user2Id } },
                team: { connect: { id: team1.id } },
            },
        });

        // Seed a label for updateOne tests
        label1Id = uuid();
        await DbProvider.get().label.create({
            data: {
                id: label1Id,
                label: "Test Label",
                color: "#123456",
                ownedByUser: { connect: { id: user1Id } },
            },
        });
    });

    after(async () => {
        // Clean up and restore logger
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().issue_translation.deleteMany();
        await DbProvider.get().issue.deleteMany();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().team.deleteMany();
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own issue for authenticated user", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: issueUser1.id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(issueUser1.id);
            });

            it("returns another user's issue", async () => {
                const user = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: issueUser2.id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result.id).to.equal(issueUser2.id);
            });

            it("returns issue when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: issueUser1.id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result.id).to.equal(issueUser1.id);
            });

            it("returns issue with API key public read", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: FindByIdInput = { id: issueUser2.id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result.id).to.equal(issueUser2.id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns all issues for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: IssueSearchInput = { take: 10 };
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                const ids = result.edges.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([issueUser1.id, issueUser2.id].sort());
            });

            it("returns all issues when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: IssueSearchInput = { take: 10 };
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                const ids = result.edges.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([issueUser1.id, issueUser2.id].sort());
            });

            it("returns issues with API key public read", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: IssueSearchInput = { take: 10 };
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                const ids = result.edges.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([issueUser1.id, issueUser2.id].sort());
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates an issue for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const newIssueId = uuid();
                const input: IssueCreateInput = { id: newIssueId, issueFor: "Team", forConnect: team1.id };
                const result = await issue.createOne({ input }, { req, res }, issue_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newIssueId);
                expect(result.status).to.equal(IssueStatus.Open);
            });

            it("API key with write permissions can create issue", async () => {
                const permissions = { [ApiKeyPermission.WritePrivate]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const newIssueId = uuid();
                const input: IssueCreateInput = { id: newIssueId, issueFor: "Team", forConnect: team1.id };
                const result = await issue.createOne({ input }, { req, res }, issue_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newIssueId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create issue", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: IssueCreateInput = { id: uuid(), issueFor: "Team", forConnect: team1.id };
                try {
                    await issue.createOne({ input }, { req, res }, issue_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });

            it("API key without write permissions cannot create issue", async () => {
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: IssueCreateInput = { id: uuid(), issueFor: "Team", forConnect: team1.id };
                try {
                    await issue.createOne({ input }, { req, res }, issue_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("allows admin to connect a label to an issue", async () => {
                // Ensure admin user exists
                await DbProvider.get().user.upsert({
                    where: { id: SEEDED_IDS.User.Admin },
                    update: {},
                    create: {
                        id: SEEDED_IDS.User.Admin,
                        name: "Admin User",
                        handle: "admin",
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                        auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
                    },
                });
                const admin = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(admin);
                const input: IssueUpdateInput = { id: issueUser1.id, labelsConnect: [label1Id] };
                const result = await issue.updateOne({ input }, { req, res }, issue_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(issueUser1.id);
                expect(result.labels.map(l => l.id)).to.include(label1Id);
            });
        });

        describe("invalid", () => {
            it("denies update for non-admin user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: IssueUpdateInput = { id: issueUser2.id, labelsConnect: [label1Id] };
                try {
                    await issue.updateOne({ input }, { req, res }, issue_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });

            it("denies update for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: IssueUpdateInput = { id: issueUser1.id, labelsConnect: [label1Id] };
                try {
                    await issue.updateOne({ input }, { req, res }, issue_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });

            it("throws when updating non-existent issue as admin", async () => {
                await DbProvider.get().user.upsert({
                    where: { id: SEEDED_IDS.User.Admin },
                    update: {},
                    create: {
                        id: SEEDED_IDS.User.Admin,
                        name: "Admin User",
                        handle: "admin",
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                        auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
                    },
                });
                const admin = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(admin);
                const input: IssueUpdateInput = { id: uuid(), labelsConnect: [label1Id] };
                try {
                    await issue.updateOne({ input }, { req, res }, issue_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });
        });
    });

    describe("closeOne", () => {
        it("throws NotImplemented error", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: IssueCloseInput = { id: issueUser1.id, status: IssueStatus.ClosedResolved };
            try {
                await issue.closeOne({ input }, { req, res }, issue_closeOne);
                expect.fail("Expected NotImplemented error");
            } catch (err: any) {
                // CustomError sets code to errorCode and message to `${errorCode}: ${trace}`
                expect(err).to.have.property("code", "NotImplemented");
                expect(err.message).to.match(/^NotImplemented/);
            }
        });
    });
}); 
