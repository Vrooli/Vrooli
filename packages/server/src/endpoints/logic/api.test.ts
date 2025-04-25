/* eslint-disable @typescript-eslint/no-non-null-assertion */
import { Api, ApiCreateInput, ApiSearchInput, ApiUpdateInput, FindByIdInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { api_createOne } from "../generated/api_createOne.js";
import { api_findMany } from "../generated/api_findMany.js";
import { api_findOne } from "../generated/api_findOne.js";
import { api_updateOne } from "../generated/api_updateOne.js";
import { api } from "./api.js";

describe("EndpointsApi", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    // Generate unique IDs for users and APIs
    const user1Id = uuid();
    const user2Id = uuid();
    const publicApiId1 = uuid();
    const publicApiId2 = uuid();
    const privateApiId1 = uuid();
    const privateApiId2 = uuid();

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Reset Redis and truncate relevant tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed two users for authentication and ownership tests
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

        // Seed two public APIs
        await DbProvider.get().api.create({
            data: {
                id: publicApiId1,
                isPrivate: false,
                permissions: JSON.stringify({}),
                versions: { create: [{ id: uuid(), callLink: "https://public-api-1.com", translations: { create: [{ id: uuid(), language: "en", name: "Public API 1" }] } }] },
            },
        });
        await DbProvider.get().api.create({
            data: {
                id: publicApiId2,
                isPrivate: false,
                permissions: JSON.stringify({}),
                versions: { create: [{ id: uuid(), callLink: "https://public-api-2.com", translations: { create: [{ id: uuid(), language: "en", name: "Public API 2" }] } }] },
            },
        });

        // Seed two private APIs with different owners
        await DbProvider.get().api.create({
            data: {
                id: privateApiId1,
                isPrivate: true,
                permissions: JSON.stringify({}),
                ownedByUser: { connect: { id: user1Id } },
                versions: { create: [{ id: uuid(), callLink: "https://private-api-1.com", translations: { create: [{ id: uuid(), language: "en", name: "Private API 1" }] } }] },
            },
        });
        await DbProvider.get().api.create({
            data: {
                id: privateApiId2,
                isPrivate: true,
                permissions: JSON.stringify({}),
                ownedByUser: { connect: { id: user2Id } },
                versions: { create: [{ id: uuid(), callLink: "https://private-api-2.com", translations: { create: [{ id: uuid(), language: "en", name: "Private API 2" }] } }] },
            },
        });
    });

    after(async () => {
        // Clean up database and restore logger
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns public API for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: FindByIdInput = { id: publicApiId1 };
                const result = await api.findOne({ input }, { req, res }, api_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(publicApiId1);
            });

            it("returns private API owned by user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: FindByIdInput = { id: privateApiId1 };
                const result = await api.findOne({ input }, { req, res }, api_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(privateApiId1);
            });

            it("returns public API for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: publicApiId2 };
                const result = await api.findOne({ input }, { req, res }, api_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(publicApiId2);
            });

            it("returns public API for API key with public read", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: publicApiId1 };
                const result = await api.findOne({ input }, { req, res }, api_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(publicApiId1);
            });
        });

        describe("invalid", () => {
            it("throws error for non-existent API id", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: uuid() };
                try {
                    await api.findOne({ input }, { req, res }, api_findOne);
                    expect.fail("Expected an error for non-existent API");
                } catch (err) {
                    console.log("[endpoint testing error debug] findOne non-existent error:", err);
                }
            });

            it("throws error for private API not owned by user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: FindByIdInput = { id: privateApiId2 };
                try {
                    await api.findOne({ input }, { req, res }, api_findOne);
                    expect.fail("Expected an error for unauthorized private API");
                } catch (err) {
                    console.log("[endpoint testing error debug] findOne private not owned error:", err);
                }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns public and own private APIs for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: ApiSearchInput = { take: 10 };
                const result = await api.findMany({ input }, { req, res }, api_findMany);
                expect(result).to.not.be.null;
                const ids = result.edges!.map(e => e!.node!.id).sort();
                expect(ids).to.include.members([publicApiId1, publicApiId2, privateApiId1]);
                expect(ids).to.not.include(privateApiId2);
            });

            it("returns only public APIs for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ApiSearchInput = { take: 10 };
                const result = await api.findMany({ input }, { req, res }, api_findMany);
                expect(result).to.not.be.null;
                const ids = result.edges!.map(e => e!.node!.id).sort();
                expect(ids).to.deep.equal([publicApiId1, publicApiId2].sort());
            });

            it("returns all APIs for API key with ReadPrivate permission", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: ApiSearchInput = { take: 10 };
                const result = await api.findMany({ input }, { req, res }, api_findMany);
                const expectedIds = [
                    publicApiId1,
                    publicApiId2,
                    privateApiId1,
                ].sort();
                expect(result.edges!.map(e => e!.node!.id).sort()).to.deep.equal(expectedIds);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a new API for API key with private write permissions", async () => {
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const permissions = mockWritePrivatePermissions();
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newApiId = uuid();
                const versionId = uuid();
                const translationId = uuid();
                const input: ApiCreateInput = {
                    id: newApiId,
                    isPrivate: false,
                    permissions: JSON.stringify({}),
                    ownedByUserConnect: testUser.id,
                    versionsCreate: [
                        {
                            id: versionId,
                            callLink: "https://test-api.com",
                            isPrivate: false,
                            versionLabel: "1.0.0",
                            translationsCreate: [
                                {
                                    id: translationId,
                                    language: "en",
                                    name: "Test API",
                                },
                            ],
                        },
                    ],
                };
                const result = await api.createOne({ input }, { req, res }, api_createOne) as Api;
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newApiId);
                expect(result.versions).to.have.lengthOf(1);
                expect(result.versions[0].callLink).to.equal("https://test-api.com");
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ApiCreateInput = { id: uuid(), isPrivate: false, permissions: JSON.stringify({}), versionsCreate: [] };
                try {
                    await api.createOne({ input }, { req, res }, api_createOne);
                    expect.fail("Expected an error for unauthorized create");
                } catch (err) {
                    console.log("[endpoint testing error debug] createOne not authenticated error:", err);
                }
            });

            it("throws error for API key without private write permissions", async () => {
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const permissions = mockReadPrivatePermissions();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input = { id: uuid(), isPrivate: false, permissions: JSON.stringify({}), versionsCreate: [] } as ApiCreateInput;
                try {
                    await api.createOne({ input }, { req, res }, api_createOne);
                    expect.fail("Expected an error for insufficient permissions");
                } catch (err) {
                    console.log("[endpoint testing error debug] createOne insufficient permission error:", err);
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates an existing API for API key with private write permissions", async () => {
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const permissions = mockWritePrivatePermissions();
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input = { id: privateApiId1, isPrivate: false } as ApiUpdateInput;
                const result = await api.updateOne({ input }, { req, res }, api_updateOne) as Api;
                expect(result).to.not.be.null;
                expect(result.isPrivate).to.be.false;
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ApiUpdateInput = { id: publicApiId1, isPrivate: true };
                try {
                    await api.updateOne({ input }, { req, res }, api_updateOne);
                    expect.fail("Expected an error for unauthorized update");
                } catch (err) {
                    console.log("[endpoint testing error debug] updateOne not authenticated error:", err);
                }
            });

            it("throws error for API key without private write permissions", async () => {
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const permissions = mockReadPrivatePermissions();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input = { id: publicApiId1, isPrivate: false } as ApiUpdateInput;
                try {
                    await api.updateOne({ input }, { req, res }, api_updateOne);
                    expect.fail("Expected an error for insufficient permissions");
                } catch (err) {
                    console.log("[endpoint testing error debug] updateOne insufficient permission error:", err);
                }
            });

            it("throws error for non-existent API id", async () => {
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const permissions = mockWritePrivatePermissions();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input = { id: uuid(), isPrivate: false } as ApiUpdateInput;
                try {
                    await api.updateOne({ input }, { req, res }, api_updateOne);
                    expect.fail("Expected an error for non-existent API");
                } catch (err) {
                    console.log("[endpoint testing error debug] updateOne non-existent error:", err);
                }
            });
        });
    });
}); 
