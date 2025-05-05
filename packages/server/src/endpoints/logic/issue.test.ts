import { FindByIdInput, IssueCloseInput, IssueCreateInput, IssueFor, IssueSearchInput, IssueStatus, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { issue_closeOne } from "../generated/issue_closeOne.js";
import { issue_createOne } from "../generated/issue_createOne.js";
import { issue_findMany } from "../generated/issue_findMany.js";
import { issue_findOne } from "../generated/issue_findOne.js";
import { issue } from "./issue.js";

// Generate test user IDs
const user1Id = uuid();
const user2Id = uuid();
let team1: any;
let issueUser1: any;
let issueUser2: any;

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
        await DbProvider.deleteAll();

        // Create a team for issue ownership
        team1 = await DbProvider.get().team.create({ data: { permissions: "{}" } });

        // Create two users
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
    });

    after(async () => {
        // Clean up and restore logger
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own issue for authenticated user", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: issueUser1.id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(issueUser1.id);
            });

            it("returns another user's issue", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
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
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
                const input: FindByIdInput = { id: issueUser2.id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result.id).to.equal(issueUser2.id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns all issues for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: IssueSearchInput = { take: 10 };
                const expectedIds = [
                    issueUser1.id,
                    issueUser2.id,
                ];
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns all issues when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: IssueSearchInput = { take: 10 };
                const expectedIds = [
                    issueUser1.id,
                    issueUser2.id,
                ];
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns issues with API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
                const input: IssueSearchInput = { take: 10 };
                const expectedIds = [
                    issueUser1.id,
                    issueUser2.id,
                ];
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                assertFindManyResultIds(expect, result, expectedIds);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates an issue for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const newIssueId = uuid();
                const input: IssueCreateInput = { id: newIssueId, issueFor: IssueFor.Team, forConnect: team1.id };
                const result = await issue.createOne({ input }, { req, res }, issue_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newIssueId);
                expect(result.status).to.equal(IssueStatus.Open);
            });

            it("API key with write permissions can create issue", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
                const newIssueId = uuid();
                const input: IssueCreateInput = { id: newIssueId, issueFor: IssueFor.Team, forConnect: team1.id };
                const result = await issue.createOne({ input }, { req, res }, issue_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newIssueId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create issue", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: IssueCreateInput = { id: uuid(), issueFor: IssueFor.Team, forConnect: team1.id };
                try {
                    await issue.createOne({ input }, { req, res }, issue_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });

            it("API key without write permissions cannot create issue", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
                const input: IssueCreateInput = { id: uuid(), issueFor: IssueFor.Team, forConnect: team1.id };
                try {
                    await issue.createOne({ input }, { req, res }, issue_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // expected error
                }
            });
        });
    });

    describe("closeOne", () => {
        it("throws NotImplemented error", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
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
