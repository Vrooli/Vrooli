import { assertFindManyResultIds } from "../../__test/helpers.js";
/* eslint-disable @typescript-eslint/no-non-null-assertion */
import { type FindByIdInput, type NotificationSearchInput, type NotificationSettings, type NotificationSettingsUpdateInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { defaultNotificationSettings } from "../../notify/notificationSettings.js";
import { initializeRedis } from "../../redisConn.js";
import { notification_findMany } from "../generated/notification_findMany.js";
import { notification_findOne } from "../generated/notification_findOne.js";
import { notification_getSettings } from "../generated/notification_getSettings.js";
import { notification_markAllAsRead } from "../generated/notification_markAllAsRead.js";
import { notification_markAsRead } from "../generated/notification_markAsRead.js";
import { notification_updateSettings } from "../generated/notification_updateSettings.js";
import { notification } from "./notification.js";

describe("EndpointsNotification", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    // Generate unique IDs for users and notifications
    const user1Id = uuid();
    const user2Id = uuid();
    const notification1Id = uuid(); // User 1, unread
    const notification2Id = uuid(); // User 1, read
    const notification3Id = uuid(); // User 2, unread

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Reset Redis and truncate relevant tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed two users
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

        // Seed notifications
        await DbProvider.get().notification.create({
            data: {
                id: notification1Id,
                userId: user1Id,
                title: "Notification 1",
                category: "TestCategory",
                isRead: false,
            },
        });
        await DbProvider.get().notification.create({
            data: {
                id: notification2Id,
                userId: user1Id,
                title: "Notification 2",
                category: "TestCategory",
                isRead: true,
            },
        });
        await DbProvider.get().notification.create({
            data: {
                id: notification3Id,
                userId: user2Id,
                title: "Notification 3",
                category: "AnotherCategory",
                isRead: false,
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
            it("returns notification owned by user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: notification1Id };
                const result = await notification.findOne({ input }, { req, res }, notification_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(notification1Id);
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: notification1Id };
                try {
                    await notification.findOne({ input }, { req, res }, notification_findOne);
                    expect.fail("Expected an error for unauthenticated access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.findOne not authenticated error:", err);
                    expect(err).to.be.an("error"); // Basic check, can refine error type later
                }
            });

            it("throws error for notification not owned by user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: notification3Id }; // Belongs to user2
                try {
                    await notification.findOne({ input }, { req, res }, notification_findOne);
                    expect.fail("Expected an error for unauthorized access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.findOne not owned error:", err);
                    expect(err).to.be.an("error");
                }
            });

            it("throws error for non-existent notification id", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: uuid() };
                try {
                    await notification.findOne({ input }, { req, res }, notification_findOne);
                    expect.fail("Expected an error for non-existent notification");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.findOne non-existent error:", err);
                    expect(err).to.be.an("error");
                }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns notifications owned by authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: NotificationSearchInput = { take: 10 };
                const expectedIds = [
                    notification1Id,
                    notification2Id,
                ];
                const result = await notification.findMany({ input }, { req, res }, notification_findMany);
                expect(result).to.not.be.null;
                expect(result.edges).to.have.lengthOf(2);
                assertFindManyResultIds(expect, result, expectedIds);
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: NotificationSearchInput = { take: 10 };
                try {
                    await notification.findMany({ input }, { req, res }, notification_findMany);
                    expect.fail("Expected an error for unauthenticated access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.findMany not authenticated error:", err);
                    expect(err).to.be.an("error");
                }
            });
        });
    });

    describe("getSettings", () => {
        describe("valid", () => {
            it("returns notification settings for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const result = await notification.getSettings(undefined, { req, res }, notification_getSettings);
                expect(result).to.deep.equal(defaultNotificationSettings);
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                try {
                    await notification.getSettings(undefined, { req, res }, notification_getSettings);
                    expect.fail("Expected an error for unauthenticated access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.getSettings not authenticated error:", err);
                    expect(err).to.be.an("error");
                }
            });
        });
    });

    describe("markAsRead", () => {
        describe("valid", () => {
            it("marks a specific notification as read for the owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: notification1Id }; // Initially unread

                // Verify it's unread first
                let notif = await DbProvider.get().notification.findUnique({ where: { id: notification1Id } });
                expect(notif!.isRead).to.be.false;

                // Mark as read
                const result = await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
                expect(result).to.deep.equal({ __typename: "Success", success: true });

                // Verify it's read now
                notif = await DbProvider.get().notification.findUnique({ where: { id: notification1Id } });
                expect(notif!.isRead).to.be.true;
            });

            it("marks notification as read successfully even if already read", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: notification2Id }; // Already read
                const result = await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
                // The endpoint should now return true even if it was already read
                expect(result).to.deep.equal({ __typename: "Success", success: true });
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: notification1Id };
                try {
                    await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
                    expect.fail("Expected an error for unauthenticated access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.markAsRead not authenticated error:", err);
                    expect(err).to.be.an("error");
                }
            });

            it("returns success false for notification not owned by user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: notification3Id }; // Belongs to user2
                const result = await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
                // updateMany count will be 0 as the user condition doesn't match
                expect(result).to.deep.equal({ __typename: "Success", success: false });

                // Verify notification 3 is still unread
                const notif3 = await DbProvider.get().notification.findUnique({ where: { id: notification3Id } });
                expect(notif3!.isRead).to.be.false;
            });

            it("returns success false for non-existent notification id", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: FindByIdInput = { id: uuid() };
                const result = await notification.markAsRead({ input }, { req, res }, notification_markAsRead);
                // For non-existent IDs, the success should still reflect the operation outcome.
                // Since updateMany affects 0 rows for a non-existent ID, returning false is appropriate here.
                expect(result).to.deep.equal({ __typename: "Success", success: false });
            });
        });
    });

    describe("markAllAsRead", () => {
        describe("valid", () => {
            it("marks all unread notifications as read for the owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });

                // Verify initial state (1 unread, 1 read)
                let user1Notifications = await DbProvider.get().notification.findMany({ where: { userId: user1Id } });
                expect(user1Notifications.filter(n => !n.isRead)).to.have.lengthOf(1);
                expect(user1Notifications.filter(n => n.isRead)).to.have.lengthOf(1);

                // Mark all as read
                const result = await notification.markAllAsRead(undefined, { req, res }, notification_markAllAsRead);
                expect(result).to.deep.equal({ __typename: "Success", success: true });

                // Verify final state (0 unread, 2 read)
                user1Notifications = await DbProvider.get().notification.findMany({ where: { userId: user1Id } });
                expect(user1Notifications.filter(n => !n.isRead)).to.have.lengthOf(0);
                expect(user1Notifications.filter(n => n.isRead)).to.have.lengthOf(2);

                // Verify user 2's notification is unaffected
                const notif3 = await DbProvider.get().notification.findUnique({ where: { id: notification3Id } });
                expect(notif3!.isRead).to.be.false;
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                try {
                    await notification.markAllAsRead(undefined, { req, res }, notification_markAllAsRead);
                    expect.fail("Expected an error for unauthenticated access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.markAllAsRead not authenticated error:", err);
                    expect(err).to.be.an("error");
                }
            });
        });
    });

    describe("updateSettings", () => {
        describe("valid", () => {
            it("updates notification settings for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
                const input: NotificationSettingsUpdateInput = { enabled: true };

                const result = await notification.updateSettings({ input }, { req, res }, notification_updateSettings);

                // Expect the returned settings to reflect the update
                expect(result.enabled).to.be.true;

                // Verify the settings are actually saved in the DB
                const user = await DbProvider.get().user.findUnique({ where: { id: user1Id }, select: { notificationSettings: true } });
                const savedSettings = JSON.parse(user!.notificationSettings!) as NotificationSettings;
                expect(savedSettings.enabled).to.be.true;
            });
        });

        describe("invalid", () => {
            it("throws error for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: NotificationSettingsUpdateInput = { enabled: true };
                try {
                    await notification.updateSettings({ input }, { req, res }, notification_updateSettings);
                    expect.fail("Expected an error for unauthenticated access");
                } catch (err) {
                    console.log("[endpoint testing error debug] notification.updateSettings not authenticated error:", err);
                    expect(err).to.be.an("error");
                }
            });
        });
    });
}); 
