// Tests for the Project endpoint (findOne, findMany, createOne, updateOne)
import { FindByIdOrHandleInput, ProjectCreateInput, ProjectSearchInput, ProjectUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { project_createOne } from "../generated/project_createOne.js";
import { project_findMany } from "../generated/project_findMany.js";
import { project_findOne } from "../generated/project_findOne.js";
import { project_updateOne } from "../generated/project_updateOne.js";
import { project } from "./project.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();
// Test projects
let projUser1Public: any;
let projUser1Private: any;
let projUser2Public: any;

describe("EndpointsProject", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // suppress logger
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // clear DBs
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // create users
        await DbProvider.get().user.create({ data: { id: user1Id, name: "User 1", handle: "user1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "hash" }] } } });
        await DbProvider.get().user.create({ data: { id: user2Id, name: "User 2", handle: "user2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "hash" }] } } });

        // seed projects
        projUser1Public = await DbProvider.get().project.create({
            data: {
                id: uuid(), isPrivate: false, ownedByUser: { connect: { id: user1Id } },
                handle: "proj-user1-public", hasBeenTransferred: false, hasCompleteVersion: false, isDeleted: false,
                score: 0, bookmarks: 0, views: 0, permissions: "",
            }
        });
        projUser1Private = await DbProvider.get().project.create({
            data: {
                id: uuid(), isPrivate: true, ownedByUser: { connect: { id: user1Id } },
                handle: "proj-user1-private", hasBeenTransferred: false, hasCompleteVersion: false, isDeleted: false,
                score: 0, bookmarks: 0, views: 0, permissions: "",
            }
        });
        projUser2Public = await DbProvider.get().project.create({
            data: {
                id: uuid(), isPrivate: false, ownedByUser: { connect: { id: user2Id } },
                handle: "proj-user2-public", hasBeenTransferred: false, hasCompleteVersion: false, isDeleted: false,
                score: 0, bookmarks: 0, views: 0, permissions: "",
            }
        });
    });

    after(async () => {
        // cleanup and restore
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("returns own public project by ID", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdOrHandleInput = { id: projUser1Public.id };
            const result = await project.findOne({ input }, { req, res }, project_findOne);
            expect(result.id).to.equal(projUser1Public.id);
        });

        it("returns own private project by ID", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdOrHandleInput = { id: projUser1Private.id };
            const result = await project.findOne({ input }, { req, res }, project_findOne);
            expect(result.id).to.equal(projUser1Private.id);
        });

        it("returns another user's public project by ID", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdOrHandleInput = { id: projUser2Public.id };
            const result = await project.findOne({ input }, { req, res }, project_findOne);
            expect(result.id).to.equal(projUser2Public.id);
        });

        it("API key with public read finds public project by ID", async () => {
            const perms = mockReadPublicPermissions();
            const token = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(token, perms, loggedInUserNoPremiumData);
            const input: FindByIdOrHandleInput = { id: projUser2Public.id };
            const result = await project.findOne({ input }, { req, res }, project_findOne);
            expect(result.id).to.equal(projUser2Public.id);
        });

        it("throws when finding another user's private project", async () => {
            // Simulate user2 trying to access user1's private project
            await DbProvider.get().project.update({ where: { id: projUser1Private.id }, data: { ownedByUserId: user1Id } }); // Ensure owner is user1
            const user = { ...loggedInUserNoPremiumData, id: user2Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdOrHandleInput = { id: projUser1Private.id };
            try {
                await project.findOne({ input }, { req, res }, project_findOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("findMany", () => {
        it("returns own public and private projects for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ProjectSearchInput = { take: 10, ownedByUserId: user1Id };
            const result = await project.findMany({ input }, { req, res }, project_findMany);
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.deep.equal([projUser1Public.id, projUser1Private.id].sort());
        });

        it("returns only public projects for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ProjectSearchInput = { take: 10 };
            const result = await project.findMany({ input }, { req, res }, project_findMany);
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.deep.equal([projUser1Public.id, projUser2Public.id].sort()); // only public ones
        });
    });

    describe("createOne", () => {
        it("creates a project for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const newProjId = uuid();
            const input: ProjectCreateInput = { id: newProjId, isPrivate: false, handle: "new-proj" };
            const result = await project.createOne({ input }, { req, res }, project_createOne);
            expect(result.id).to.equal(newProjId);
            expect(result.handle).to.equal("new-proj");
        });

        it("throws for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ProjectCreateInput = { id: uuid(), isPrivate: false, handle: "no-auth-proj" };
            try {
                await project.createOne({ input }, { req, res }, project_createOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("updateOne", () => {
        it("updates own project handle", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ProjectUpdateInput = { id: projUser1Public.id, handle: "updated-handle" };
            const result = await project.updateOne({ input }, { req, res }, project_updateOne);
            expect(result.id).to.equal(projUser1Public.id);
            expect(result.handle).to.equal("updated-handle");
        });

        it("throws when updating another user's project", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ProjectUpdateInput = { id: projUser2Public.id, handle: "hacked-handle" }; // Trying to update User 2's project
            try {
                await project.updateOne({ input }, { req, res }, project_updateOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });

        it("throws for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ProjectUpdateInput = { id: projUser1Public.id, handle: "no-auth-update" };
            try {
                await project.updateOne({ input }, { req, res }, project_updateOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });
}); 