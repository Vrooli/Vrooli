import { ReminderListCreateInput, ReminderListUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { reminderList_createOne } from "../generated/reminderList_createOne.js";
import { reminderList_updateOne } from "../generated/reminderList_updateOne.js";
import { reminderList } from "./reminderList.js";

// Test users
const user1Id = uuid();
const user2Id = uuid();

// ReminderList IDs for tests
const reminderListUser1Id = uuid();
const reminderListUser2Id = uuid();

describe("EndpointsReminderList", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create test users
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

        // Create ReminderLists for users
        await DbProvider.get().reminder_list.create({
            data: {
                id: reminderListUser1Id,
                created_at: new Date("2023-03-01"),
                updated_at: new Date("2023-03-01"),
                user: { connect: { id: user1Id } },
            },
        });
        await DbProvider.get().reminder_list.create({
            data: {
                id: reminderListUser2Id,
                created_at: new Date("2023-03-05"),
                updated_at: new Date("2023-03-05"),
                user: { connect: { id: user2Id } },
            },
        });
    });

    after(async function after() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Restore logger stubs
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates a reminder list for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const newListId = uuid();
                const input: ReminderListCreateInput = { id: newListId };

                const creationResult = await reminderList.createOne({ input }, { req, res }, reminderList_createOne);

                expect(creationResult).to.not.be.null;
                expect(creationResult.id).to.equal(newListId);
            });

            it("API key with write permissions can create reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const newListId = uuid();
                const input: ReminderListCreateInput = { id: newListId };

                const result = await reminderList.createOne({ input }, { req, res }, reminderList_createOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(newListId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create reminder list", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderListCreateInput = { id: uuid() };

                try {
                    await reminderList.createOne({ input }, { req, res }, reminderList_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("API key without write permissions cannot create reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderListCreateInput = { id: uuid() };

                try {
                    await reminderList.createOne({ input }, { req, res }, reminderList_createOne);
                    expect.fail("Expected an error to be thrown - write permission required");
                } catch (error) { /** Error expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates a reminder list for authenticated user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderListUpdateInput = { id: reminderListUser1Id };

                const result = await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(reminderListUser1Id);
            });

            it("API key with write permissions can update reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ReminderListUpdateInput = { id: reminderListUser1Id };

                const result = await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);

                expect(result).to.not.be.null;
                expect(result.id).to.equal(reminderListUser1Id);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderListUpdateInput = { id: reminderListUser2Id };

                try {
                    await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("not logged in user cannot update reminder list", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ReminderListUpdateInput = { id: reminderListUser1Id };

                try {
                    await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });

            it("cannot update non-existent reminder list", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: ReminderListUpdateInput = { id: uuid() };

                try {
                    await reminderList.updateOne({ input }, { req, res }, reminderList_updateOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Error expected */ }
            });
        });
    });
});
