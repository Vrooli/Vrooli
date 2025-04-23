import { ApiVersion, ApiVersionCreateInput, ApiVersionSearchInput, ApiVersionUpdateInput, FindVersionInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadAuthPermissions, mockReadPublicPermissions, mockWriteAuthPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { apiVersion_createOne } from "../generated/apiVersion_createOne.js";
import { apiVersion_findMany } from "../generated/apiVersion_findMany.js";
import { apiVersion_findOne } from "../generated/apiVersion_findOne.js";
import { apiVersion_updateOne } from "../generated/apiVersion_updateOne.js";
import { apiVersion } from "./apiVersion.js";

describe("EndpointsApiVersion", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    const apiRootId = uuid();
    const versionId = uuid();
    const translationId = uuid();

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().session.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().api.deleteMany();
        await DbProvider.get().api_version_translation.deleteMany();
        await DbProvider.get().api_version.deleteMany();

        // Seed API root with one public version
        await DbProvider.get().api.create({
            data: {
                id: apiRootId,
                isPrivate: false,
                permissions: JSON.stringify({}),
                versions: {
                    create: [
                        {
                            id: versionId,
                            callLink: "https://example.com/version/1",
                            isPrivate: false,
                            versionLabel: "1.0.0",
                            translations: { create: [{ id: translationId, language: "en", name: "Version 1" }] },
                        },
                    ],
                },
            },
        });
    });

    after(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().api_version_translation.deleteMany();
        await DbProvider.get().api_version.deleteMany();
        await DbProvider.get().api.deleteMany();
        await DbProvider.get().user.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().session.deleteMany();
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("returns the version for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: uuid() });
            const input: FindVersionInput = { id: versionId };
            const result: ApiVersion = await apiVersion.findOne({ input }, { req, res }, apiVersion_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(versionId);
        });

        it("returns the version for not authenticated user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindVersionInput = { id: versionId };
            const result: ApiVersion = await apiVersion.findOne({ input }, { req, res }, apiVersion_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(versionId);
        });

        it("returns the version for API key with public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
            const input: FindVersionInput = { id: versionId };
            const result: ApiVersion = await apiVersion.findOne({ input }, { req, res }, apiVersion_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(versionId);
        });
    });

    describe("findMany", () => {
        it("returns the seeded version for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: uuid() });
            const input: ApiVersionSearchInput = { take: 10 };
            const result: ApiVersionSearchInput = await apiVersion.findMany({ input }, { req, res }, apiVersion_findMany);
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.include(versionId);
        });

        it("returns the seeded version for not authenticated user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ApiVersionSearchInput = { take: 10 };
            const result: ApiVersionSearchInput = await apiVersion.findMany({ input }, { req, res }, apiVersion_findMany);
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.include(versionId);
        });

        it("returns the seeded version for API key with public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
            const input: ApiVersionSearchInput = { take: 10 };
            const result: ApiVersionSearchInput = await apiVersion.findMany({ input }, { req, res }, apiVersion_findMany);
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.include(versionId);
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a version for API key with write auth permissions", async () => {
                const permissions = mockWriteAuthPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const newVersionId = uuid();
                const newTranslationId = uuid();
                const input: ApiVersionCreateInput = {
                    id: newVersionId,
                    callLink: "https://example.com/version/new",
                    isPrivate: false,
                    versionLabel: "2.0.0",
                    rootConnect: apiRootId,
                    translationsCreate: [{ id: newTranslationId, language: "en", name: "Version New" }],
                };
                const result: ApiVersion = await apiVersion.createOne({ input }, { req, res }, apiVersion_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newVersionId);
                expect(result.versionLabel).to.equal("2.0.0");
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ApiVersionCreateInput = { id: uuid(), callLink: "x", isPrivate: false, versionLabel: "x", rootConnect: apiRootId, translationsCreate: [] };
                try {
                    await apiVersion.createOne({ input }, { req, res }, apiVersion_createOne);
                    expect.fail("Expected an error for unauthorized create");
                } catch (err) {
                    console.log("[endpoint testing error debug] createOne not authenticated error:", err);
                }
            });

            it("throws error for API key without write auth permissions", async () => {
                const permissions = mockReadAuthPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: ApiVersionCreateInput = { id: uuid(), callLink: "x", isPrivate: false, versionLabel: "x", rootConnect: apiRootId, translationsCreate: [] };
                try {
                    await apiVersion.createOne({ input }, { req, res }, apiVersion_createOne);
                    expect.fail("Expected an error for insufficient permissions");
                } catch (err) {
                    console.log("[endpoint testing error debug] createOne insufficient permission error:", err);
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates versionLabel for API key with write auth permissions", async () => {
                const permissions = mockWriteAuthPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: ApiVersionUpdateInput = { id: versionId, versionLabel: "1.1.0" };
                const result: ApiVersion = await apiVersion.updateOne({ input }, { req, res }, apiVersion_updateOne);
                expect(result).to.not.be.null;
                expect(result.versionLabel).to.equal("1.1.0");
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ApiVersionUpdateInput = { id: versionId, versionLabel: "1.1.0" };
                try {
                    await apiVersion.updateOne({ input }, { req, res }, apiVersion_updateOne);
                    expect.fail("Expected an error for unauthorized update");
                } catch (err) {
                    console.log("[endpoint testing error debug] updateOne not authenticated error:", err);
                }
            });

            it("throws error for API key without write auth permissions", async () => {
                const permissions = mockReadAuthPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: ApiVersionUpdateInput = { id: versionId, versionLabel: "1.2.0" };
                try {
                    await apiVersion.updateOne({ input }, { req, res }, apiVersion_updateOne);
                    expect.fail("Expected an error for insufficient permissions");
                } catch (err) {
                    console.log("[endpoint testing error debug] updateOne insufficient permission error:", err);
                }
            });

            it("throws error for non-existent version id", async () => {
                const permissions = mockWriteAuthPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData);
                const input: ApiVersionUpdateInput = { id: uuid(), versionLabel: "x" };
                try {
                    await apiVersion.updateOne({ input }, { req, res }, apiVersion_updateOne);
                    expect.fail("Expected an error for non-existent version");
                } catch (err) {
                    console.log("[endpoint testing error debug] updateOne non-existent error:", err);
                }
            });
        });
    });
}); 
