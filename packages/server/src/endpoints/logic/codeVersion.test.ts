// Tests for the CodeVersion endpoint (findOne, findMany, createOne, updateOne)
import { CodeType, CodeVersionCreateInput, CodeVersionSearchInput, CodeVersionUpdateInput, FindVersionInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { codeVersion_createOne } from "../generated/codeVersion_createOne.js";
import { codeVersion_findMany } from "../generated/codeVersion_findMany.js";
import { codeVersion_findOne } from "../generated/codeVersion_findOne.js";
import { codeVersion_updateOne } from "../generated/codeVersion_updateOne.js";
import { codeVersion } from "./codeVersion.js";

// Test user, code and version records
const user1Id = uuid();
let code1: any;
let version1: any;

describe("EndpointsCodeVersion", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // suppress logger
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // reset DB and Redis
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // create a user
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "Test User",
                handle: "test-user",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            }
        });

        // create a code record to be root
        code1 = await DbProvider.get().code.create({
            data: {
                id: uuid(),
                isPrivate: false,
                hasBeenTransferred: false,
                hasCompleteVersion: false,
                isDeleted: false,
                score: 0,
                bookmarks: 0,
                views: 0,
                permissions: "",
            }
        });

        // seed a version record
        version1 = await DbProvider.get().code_version.create({
            data: {
                id: uuid(),
                codeLanguage: "typescript",
                codeType: CodeType.DataConvert,
                content: "initial content",
                default: null,
                isLatest: true,
                isLatestPublic: true,
                isComplete: false,
                isDeleted: false,
                root: { connect: { id: code1.id } },
                versionIndex: 0,
                versionLabel: "1.0.0",
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
        it("returns version by id for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindVersionInput = { id: version1.id };
            const result = await codeVersion.findOne({ input }, { req, res }, codeVersion_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(version1.id);
        });

        it("returns version by id for API key public read", async () => {
            const perms = mockReadPublicPermissions();
            const token = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(token, perms, loggedInUserNoPremiumData);
            const input: FindVersionInput = { id: version1.id };
            const result = await codeVersion.findOne({ input }, { req, res }, codeVersion_findOne);
            expect(result.id).to.equal(version1.id);
        });

        it("throws when not found", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: FindVersionInput = { id: uuid() };
            try {
                await codeVersion.findOne({ input }, { req, res }, codeVersion_findOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("findMany", () => {
        it("returns list of versions for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: CodeVersionSearchInput = { take: 10 };
            const result = await codeVersion.findMany({ input }, { req, res }, codeVersion_findMany);
            expect(result.edges).to.not.be.undefined;
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).to.include(version1.id);
        });

        it("returns list of versions for API key public read", async () => {
            const perms = mockReadPublicPermissions();
            const token = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(token, perms, loggedInUserNoPremiumData);
            const input: CodeVersionSearchInput = { take: 10 };
            const result = await codeVersion.findMany({ input }, { req, res }, codeVersion_findMany);
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).to.include(version1.id);
        });
    });

    describe("createOne", () => {
        it("creates a version for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const newVerId = uuid();
            const input: CodeVersionCreateInput = {
                id: newVerId,
                codeLanguage: "js",
                codeType: CodeType.DataConvert,
                content: "new content",
                isPrivate: false,
                rootConnect: code1.id,
                versionLabel: "1.0.1",
            };
            const result = await codeVersion.createOne({ input }, { req, res }, codeVersion_createOne);
            expect(result.id).to.equal(newVerId);
            expect(result.content).to.equal("new content");
        });

        it("throws for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: CodeVersionCreateInput = {
                id: uuid(), codeLanguage: "js", codeType: CodeType.DataConvert,
                content: "fail content", isPrivate: false, rootConnect: code1.id, versionLabel: "1.0.2",
            };
            try {
                await codeVersion.createOne({ input }, { req, res }, codeVersion_createOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("updateOne", () => {
        it("updates versionNotes for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: CodeVersionUpdateInput = { id: version1.id, versionNotes: "updated note" };
            const result = await codeVersion.updateOne({ input }, { req, res }, codeVersion_updateOne);
            expect(result.id).to.equal(version1.id);
            expect(result.versionNotes).to.equal("updated note");
        });

        it("throws for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: CodeVersionUpdateInput = { id: version1.id, versionNotes: "no auth" };
            try {
                await codeVersion.updateOne({ input }, { req, res }, codeVersion_updateOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });
}); 