import { type FindByIdInput, type NotificationSearchInput, type NotificationSettingsUpdateInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { defaultNotificationSettings } from "../../notify/notificationSettings.js";
import { notification_findMany } from "../generated/notification_findMany.js";
import { notification_findOne } from "../generated/notification_findOne.js";
import { notification_getSettings } from "../generated/notification_getSettings.js";
import { notification_markAllAsRead } from "../generated/notification_markAllAsRead.js";
import { notification_markAsRead } from "../generated/notification_markAsRead.js";
import { notification_updateSettings } from "../generated/notification_updateSettings.js";
import { notification } from "./notification.js";
// Import database fixtures for seeding
import { seedNotifications } from "../../__test/fixtures/db/notificationFixtures.js";
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsNotification", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user","user_auth","email","session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn('Test cleanup incomplete:', orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());
    }););

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns own notification for authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: FindByIdInput = { id: notificationData.notifications[0].id };
            const result = await notification.findOne({ input }, { req, res }, notification_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(notificationData.notifications[0].id);
            expect(result.userId).toEqual(testUsers[0].id);
        });

        it("does not return another user's notification", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id.toString(),
            });
            const input: FindByIdInput = { id: notificationData.notifications[0].id };

            await expect(async () => {
                await notification.findOne({ input }, { req, res }, notification_findOne);
            }).rejects.toThrow();
        });

        it("throws error when not authenticated", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: notificationData.notifications[0].id };

            await expect(async () => {
                await notification.findOne({ input }, { req, res }, notification_findOne);
            }).rejects.toThrow();
        });
    });

    describe("findMany", () => {
        it("returns own notifications for authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: NotificationSearchInput = { take: 10 };
            const result = await notification.findMany({ input }, { req, res }, notification_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            expect(result.edges.length).toBe(3); // User 1 has 3 notifications

            // Verify all returned notifications belong to the user
            result.edges.forEach((edge: any) => {
                expect(edge.node.userId).toEqual(testUsers[0].id);
            });
        });

        it("filters by read status", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });
            const input: NotificationSearchInput = {
                isRead: false,
                take: 10,
            };
            const result = await notification.findMany({ input }, { req, res }, notification_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);

            // Should return only unread notifications
            result.edges.forEach((edge: any) => {
                expect(edge.node.isRead).toBe(false);
            });
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: NotificationSearchInput = { take: 10 };

            await expect(async () => {
                await notification.findMany({ input }, { req, res }, notification_findMany);
            }).rejects.toThrow();
        });
    });

    describe("markAsRead", () => {
        it("marks notification as read for authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            // Find an unread notification
            const unreadNotification = notificationData.notifications.find((n: any) => !n.isRead);
            const input: FindByIdInput = { id: unreadNotification.id };

            const result = await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
            expect(result).not.toBeNull();
            expect(result.isRead).toBe(true);
        });

        it("does not mark another user's notification", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id.toString(),
            });
            const input: FindByIdInput = { id: notificationData.notifications[0].id };

            await expect(async () => {
                await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
            }).rejects.toThrow();
        });

        it("throws error when not authenticated", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: notificationData.notifications[0].id };

            await expect(async () => {
                await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
            }).rejects.toThrow();
        });
    });

    describe("markAllAsRead", () => {
        it("marks all notifications as read for authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            const result = await notification.markAllAsRead({}, { req, res }, notification_markAllAsRead);
            expect(result).not.toBeNull();
            expect(result.success).toBe(true);

            // Verify all notifications are marked as read
            const notifications = await DbProvider.get().notification.findMany({
                where: { userId: testUsers[0].id },
            });
            notifications.forEach((n: any) => {
                expect(n.isRead).toBe(true);
            });
        });

        it("does not affect other users' notifications", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures for user 1
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            // Add notifications for second user
            await seedNotifications(DbProvider.get(), {
                userId: testUsers[1].id,
                count: 2,
                categories: ["Update", "Alert"],
                withRead: false,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            await notification.markAllAsRead({}, { req, res }, notification_markAllAsRead);

            // Check that user 2's notifications are still unread
            const user2Notifications = await DbProvider.get().notification.findMany({
                where: { userId: testUsers[1].id },
            });
            user2Notifications.forEach((n: any) => {
                expect(n.isRead).toBe(false);
            });
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();

            await expect(async () => {
                await notification.markAllAsRead({}, { req, res }, notification_markAllAsRead);
            }).rejects.toThrow();
        });
    });

    describe("getSettings", () => {
        it("returns notification settings for authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            const result = await notification.getSettings({}, { req, res }, notification_getSettings);
            expect(result).not.toBeNull();
            expect(result).toHaveProperty("disabled");
            expect(result).toHaveProperty("categories");
        });

        it("returns default settings if user has none", async () => {
            // Create a new user without settings
            const newUser = await DbProvider.get().user.create({
                data: UserDbFactory.createMinimal({ 
                    id: generatePK(),
                    name: "New User", 
                }),
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: newUser.id.toString(),
            });

            const result = await notification.getSettings({}, { req, res }, notification_getSettings);
            expect(result).not.toBeNull();
            expect(result).toEqual(defaultNotificationSettings);
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();

            await expect(async () => {
                await notification.getSettings({}, { req, res }, notification_getSettings);
            }).rejects.toThrow();
        });
    });

    describe("updateSettings", () => {
        it("updates notification settings for authenticated user", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            const input: NotificationSettingsUpdateInput = {
                disabled: true,
                categories: {
                    Update: {
                        enabled: false,
                        inApp: false,
                        email: false,
                        push: false,
                    },
                },
            };

            const result = await notification.updateSettings({ input }, { req, res }, notification_updateSettings);
            expect(result).not.toBeNull();
            expect(result.disabled).toBe(true);
            expect(result.categories.Update.enabled).toBe(false);
        });

        it("partially updates settings", async () => {
            // Seed test users using database fixtures
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

            // Seed notifications using database fixtures
            const notificationData = await seedNotifications(DbProvider.get(), {
                userId: testUsers[0].id,
                count: 3,
                categories: ["Update", "Reminder", "Alert"],
                withRead: true, // Mix of read and unread
                withSubscriptions: true,
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id.toString(),
            });

            const input: NotificationSettingsUpdateInput = {
                categories: {
                    Alert: {
                        email: true,
                        push: true,
                    },
                },
            };

            const result = await notification.updateSettings({ input }, { req, res }, notification_updateSettings);
            expect(result).not.toBeNull();
            expect(result.categories.Alert.email).toBe(true);
            expect(result.categories.Alert.push).toBe(true);
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();

            const input: NotificationSettingsUpdateInput = {
                disabled: true,
            };

            await expect(async () => {
                await notification.updateSettings({ input }, { req, res }, notification_updateSettings);
            }).rejects.toThrow();
        });
    });
});
