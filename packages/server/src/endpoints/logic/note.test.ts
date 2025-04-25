import { FindByIdInput, NoteCreateInput, NoteSearchInput, NoteUpdateInput, SEEDED_IDS, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { note_createOne } from "../generated/note_createOne.js";
import { note_findMany } from "../generated/note_findMany.js";
import { note_findOne } from "../generated/note_findOne.js";
import { note_updateOne } from "../generated/note_updateOne.js";
import { note } from "./note.js";

// Unique user IDs for testing
const user1Id = uuid();
const user2Id = uuid();

let note1: any;
let note2: any;

describe("EndpointsNote", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Suppress logger output during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Reset Redis and truncate tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

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

        // Ensure admin user exists for update tests
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

        // Seed a public note for user1
        const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
        const note1Id = uuid();
        const version1Id = uuid();
        const translation1Id = uuid();
        const input1: NoteCreateInput = {
            id: note1Id,
            isPrivate: false,
            versionsCreate: [
                {
                    id: version1Id,
                    isPrivate: false,
                    versionLabel: "v1",
                    translationsCreate: [{ id: translation1Id, language: "en", name: "Initial version" }],
                },
            ],
        };
        note1 = await note.createOne({ input: input1 }, { req: req1, res: res1 }, note_createOne);

        // Seed a private note for user2
        const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
        const note2Id = uuid();
        const version2Id = uuid();
        const translation2Id = uuid();
        const input2: NoteCreateInput = {
            id: note2Id,
            isPrivate: true,
            versionsCreate: [
                {
                    id: version2Id,
                    isPrivate: true,
                    versionLabel: "v1",
                    translationsCreate: [{ id: translation2Id, language: "en", name: "Private version" }],
                },
            ],
        };
        note2 = await note.createOne({ input: input2 }, { req: req2, res: res2 }, note_createOne);
    });

    after(async () => {
        // Cleanup and restore logger stubs
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("returns note by id for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
            const input: FindByIdInput = { id: note1.id };
            const result = await note.findOne({ input }, { req, res }, note_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(note1.id);
        });

        it("returns note by id when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: note2.id };
            const result = await note.findOne({ input }, { req, res }, note_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(note2.id);
        });

        it("returns note by id with API key public read", async () => {
            const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, testUser);
            const input: FindByIdInput = { id: note1.id };
            const result = await note.findOne({ input }, { req, res }, note_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(note1.id);
        });
    });

    describe("findMany", () => {
        it("returns all notes without filters for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: NoteSearchInput = { take: 10 };
            const result = await note.findMany({ input }, { req, res }, note_findMany);
            expect(result).to.not.be.null;
            expect(result.edges).to.be.an("array");
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.deep.equal([note1.id, note2.id].sort());
        });

        it("returns notes without filters for not authenticated user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: NoteSearchInput = { take: 10 };
            const result = await note.findMany({ input }, { req, res }, note_findMany);
            expect(result).to.not.be.null;
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.deep.equal([note1.id, note2.id].sort());
        });

        it("returns notes without filters for API key public read", async () => {
            const testUser = { ...loggedInUserNoPremiumData, id: user2Id };
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, testUser);
            const input: NoteSearchInput = { take: 10 };
            const result = await note.findMany({ input }, { req, res }, note_findMany);
            expect(result).to.not.be.null;
            const ids = result.edges!.map(e => e!.node!.id).sort();
            expect(ids).to.deep.equal([note1.id, note2.id].sort());
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a note for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const newNoteId = uuid();
                const versionId = uuid();
                const translationId = uuid();
                const input: NoteCreateInput = {
                    id: newNoteId,
                    isPrivate: false,
                    versionsCreate: [
                        { id: versionId, isPrivate: false, versionLabel: "v1", translationsCreate: [{ id: translationId, language: "en", name: "Test version" }] },
                    ],
                };
                const result = await note.createOne({ input }, { req, res }, note_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newNoteId);
            });

            it("API key with write permissions can create note", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user2Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const newNoteId = uuid();
                const versionId = uuid();
                const translationId = uuid();
                const input: NoteCreateInput = {
                    id: newNoteId,
                    isPrivate: true,
                    versionsCreate: [
                        { id: versionId, isPrivate: true, versionLabel: "v1", translationsCreate: [{ id: translationId, language: "en", name: "API version" }] },
                    ],
                };
                const result = await note.createOne({ input }, { req, res }, note_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newNoteId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create note", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: NoteCreateInput = {
                    id: uuid(),
                    isPrivate: false,
                    versionsCreate: [{ id: uuid(), isPrivate: false, versionLabel: "x", translationsCreate: [{ id: uuid(), language: "en", name: "x" }] }],
                };
                try {
                    await note.createOne({ input }, { req, res }, note_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });

            it("API key without write permissions cannot create note", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: NoteCreateInput = {
                    id: uuid(),
                    isPrivate: true,
                    versionsCreate: [{ id: uuid(), isPrivate: true, versionLabel: "x", translationsCreate: [{ id: uuid(), language: "en", name: "x" }] }],
                };
                try {
                    await note.createOne({ input }, { req, res }, note_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("allows owner to update a note", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: NoteUpdateInput = { id: note1.id, isPrivate: true };
                const result = await note.updateOne({ input }, { req, res }, note_updateOne);
                expect(result).to.not.be.null;
                expect(result.isPrivate).to.equal(true);
            });
        });

        describe("invalid", () => {
            it("denies update for non-owner user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
                const input: NoteUpdateInput = { id: note1.id, isPrivate: false };
                try {
                    await note.updateOne({ input }, { req, res }, note_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });

            it("denies update for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: NoteUpdateInput = { id: note1.id, isPrivate: false };
                try {
                    await note.updateOne({ input }, { req, res }, note_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });

            it("throws when updating non-existent note as admin", async () => {
                const adminUser = { ...loggedInUserNoPremiumData, id: SEEDED_IDS.User.Admin };
                const { req, res } = await mockAuthenticatedSession(adminUser);
                const input: NoteUpdateInput = { id: uuid(), isPrivate: false };
                try {
                    await note.updateOne({ input }, { req, res }, note_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch {
                    // Error expected
                }
            });
        });
    });
});
