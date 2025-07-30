import { type ChatCreateInput, type ChatSearchInput, type FindByPublicIdInput, generatePK, generatePublicId, SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { seedTestChat } from "../../__test/fixtures/db/chatFixtures.js";
import { UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { chat_createOne } from "../generated/chat_createOne.js";
import { chat_findMany } from "../generated/chat_findMany.js";
import { chat_findOne } from "../generated/chat_findOne.js";
import { chat } from "./chat.js";
// Import validation fixtures for API input testing
import { chatTestDataFactory } from "@vrooli/shared";
// Import new cleanup helpers
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsChat", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests - chat tests need chat system cleanup
        await cleanupGroups.chat(DbProvider.get());

        // Create admin user that some tests expect
        const prisma = DbProvider.get();
        await prisma.user.create({
            data: {
                id: generatePK(),
                publicId: SEEDED_PUBLIC_IDS.Admin, // Use the correct admin public ID
                name: "Admin",
                handle: "__admin__",
                isPrivate: false,
                theme: "light",
            },
        });
    });

    afterEach(async () => {
        // Perform cleanup using dependency-ordered cleanup helpers
        await cleanupGroups.chat(DbProvider.get());

        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["chat", "chat_message", "chat_participants", "chat_invite", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Chat test cleanup incomplete:", orphans);
        }
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns chat by id for any authenticated user", async () => {
            // Create test users
            const testUser1 = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User 1",
                    handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                }),
            });
            const testUser2 = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User 2",
                    handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                }),
            });
            const testUser3 = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User 3",
                    handle: "test-user-3-" + Math.floor(Math.random() * 1000),
                }),
            });

            // Create a private chat
            const privateChat = await seedTestChat(DbProvider.get(), {
                userIds: [testUser1.id, testUser2.id],
                isPrivate: true,
                withMessages: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUser3.id.toString(),
            });
            const input: FindByPublicIdInput = { publicId: privateChat.publicId };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(privateChat.id);
        });

        it("creates a chat for authenticated user", async () => {
            // Create test user
            const testUser = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User",
                    handle: "test-user-" + Math.floor(Math.random() * 1000),
                }),
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUser.id.toString(),
            });

            // Use validation fixtures for consistent test data
            const input: ChatCreateInput = chatTestDataFactory.createMinimal({
                openToAnyoneWithInvite: true,
            });

            const result = await chat.createOne({ input }, { req, res }, chat_createOne);
            expect(result).not.toBeNull();
            expect(result.id).toBeDefined();
            expect(result.openToAnyoneWithInvite).toBe(true);
        });

        it("returns chat by id when not authenticated", async () => {
            // Create test user
            const testUser = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User",
                    handle: "test-user-" + Math.floor(Math.random() * 1000),
                }),
            });

            // Create a public chat
            const publicChat = await seedTestChat(DbProvider.get(), {
                userIds: [testUser.id],
                isPrivate: false,
                withInvites: true,
            });

            const { req, res } = await mockLoggedOutSession();
            const input: FindByPublicIdInput = { publicId: publicChat.publicId };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(publicChat.id);
        });
    });

    describe("findMany", () => {
        it("returns chats without filters for any authenticated user", async () => {
            // Create test users
            const testUser1 = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User 1",
                    handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                }),
            });
            const testUser2 = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: generatePK(),
                    name: "Test User 2",
                    handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                }),
            });

            // Create chats
            const privateChat = await seedTestChat(DbProvider.get(), {
                userIds: [testUser1.id, testUser2.id],
                isPrivate: true,
                withMessages: true,
            });
            const publicChat = await seedTestChat(DbProvider.get(), {
                userIds: [testUser2.id],
                isPrivate: false,
                withInvites: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUser1.id.toString(),
            });
            const input: ChatSearchInput = { take: 10 };
            const expectedIds = [privateChat.id, publicChat.id];
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            assertFindManyResultIds(expect, result, expectedIds);
        });
    });

    describe("createOne", () => {
        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();

            // Use validation fixtures for consistent test data
            const input: ChatCreateInput = chatTestDataFactory.createMinimal({
                openToAnyoneWithInvite: false,
            });

            await expect(async () => {
                await chat.createOne({ input }, { req, res }, chat_createOne);
            }).rejects.toThrow();
        });
    });
});
