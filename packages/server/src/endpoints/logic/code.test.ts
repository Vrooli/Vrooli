// Tests for the Code endpoint (findOne, findMany, createOne, updateOne)
import { CodeCreateInput, CodeSearchInput, CodeUpdateInput, FindByIdInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { code_createOne } from "../generated/code_createOne.js";
import { code_findMany } from "../generated/code_findMany.js";
import { code_findOne } from "../generated/code_findOne.js";
import { code_updateOne } from "../generated/code_updateOne.js";
import { code } from "./code.js";

// Test user and code record
const user1Id = uuid();
let code1: any;

describe("EndpointsCode", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // stub logger
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // reset DB
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // seed user
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
            },
        });

        // seed a code record
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
            },
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
        it("returns code by id for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdInput = { id: code1.id };
            const result = await code.findOne({ input }, { req, res }, code_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(code1.id);
        });

        it("returns code by id for API key with public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
            const input: FindByIdInput = { id: code1.id };
            const result = await code.findOne({ input }, { req, res }, code_findOne);
            expect(result.id).to.equal(code1.id);
        });

        it("throws when code not found", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdInput = { id: uuid() };
            try {
                await code.findOne({ input }, { req, res }, code_findOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("findMany", () => {
        it("returns list of codes for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: CodeSearchInput = { take: 10 };
            const result = await code.findMany({ input }, { req, res }, code_findMany);
            expect(result.edges).to.not.be.undefined;
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).to.include(code1.id);
        });

        it("returns list of codes for API key with public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
            const input: CodeSearchInput = { take: 10 };
            const result = await code.findMany({ input }, { req, res }, code_findMany);
            expect(result.edges!.map(e => e!.node!.id)).to.include(code1.id);
        });
    });

    describe("createOne", () => {
        it("creates a code record for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const newCodeId = uuid();
            const input: CodeCreateInput = { id: newCodeId, isPrivate: false };
            const result = await code.createOne({ input }, { req, res }, code_createOne);
            expect(result.id).to.equal(newCodeId);
        });

        it("throws for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: CodeCreateInput = { id: uuid(), isPrivate: false };
            try {
                await code.createOne({ input }, { req, res }, code_createOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("updateOne", () => {
        it("updates a code record for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: CodeUpdateInput = { id: code1.id, data: "new content" };
            const result = await code.updateOne({ input }, { req, res }, code_updateOne);
            expect(result.id).to.equal(code1.id);
            expect(result.data).to.equal("new content");
        });

        it("throws for logged out user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: CodeUpdateInput = { id: code1.id, data: "no auth" };
            try {
                await code.updateOne({ input }, { req, res }, code_updateOne);
                expect.fail("Expected error");
            } catch (err) {
                // expected
            }
        });
    });
}); 