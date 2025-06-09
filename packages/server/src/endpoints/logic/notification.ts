import { type FindByIdInput, type Notification, type NotificationSearchInput, type NotificationSearchResult, type NotificationSettings, type NotificationSettingsUpdateInput, type Success, VisibilityType } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { defaultNotificationSettings, updateNotificationSettings } from "../../notify/notificationSettings.js";
import { type ApiEndpoint } from "../../types.js";
import { parseJsonOrDefault } from "../../utils/objectTools.js";
import { createStandardCrudEndpoints, PermissionPresets } from "../helpers/endpointFactory.js";

export type EndpointsNotification = {
    findOne: ApiEndpoint<FindByIdInput, Notification>;
    findMany: ApiEndpoint<NotificationSearchInput, NotificationSearchResult>;
    getSettings: ApiEndpoint<undefined, NotificationSettings>;
    markAsRead: ApiEndpoint<FindByIdInput, Success>;
    markAllAsRead: ApiEndpoint<undefined, Success>;
    updateSettings: ApiEndpoint<NotificationSettingsUpdateInput, NotificationSettings>;
}

export const notification: EndpointsNotification = createStandardCrudEndpoints({
    objectType: "Notification",
    endpoints: {
        findOne: {
            rateLimit: { maxUser: 10_000 },
            permissions: PermissionPresets.READ_PRIVATE,
        },
        findMany: {
            rateLimit: { maxUser: 10_000 },
            permissions: PermissionPresets.READ_PRIVATE,
            visibility: VisibilityType.Own,
        },
    },
    customEndpoints: {
        getSettings: async (_, { req }) => {
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1_000, req });
            RequestService.assertRequestFrom(req, { hasReadPrivatePermissions: true });
            const user = await DbProvider.get().user.findUnique({
                where: { id: BigInt(id) },
                select: { notificationSettings: true },
            });
            if (!user) throw new CustomError("0402", "InternalError");
            return parseJsonOrDefault<NotificationSettings>(user.notificationSettings, defaultNotificationSettings);
        },
        markAsRead: async (wrapped, { req }) => {
            const input = wrapped?.input;
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 10_000, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            const { count } = await DbProvider.get().notification.updateMany({
                where: { AND: [{ user: { id: BigInt(userId) } }, { id: BigInt(input.id) }] },
                data: { isRead: true },
            });
            return { __typename: "Success" as const, success: count > 0 };
        },
        markAllAsRead: async (_, { req }) => {
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 10_000, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            await DbProvider.get().notification.updateMany({
                where: { AND: [{ user: { id: BigInt(userId) } }, { isRead: false }] },
                data: { isRead: true },
            });
            return { __typename: "Success" as const, success: true };
        },
        updateSettings: async (wrapped, { req }) => {
            const input = wrapped?.input;
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1_000, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            return updateNotificationSettings(input, id);
        },
    },
});
