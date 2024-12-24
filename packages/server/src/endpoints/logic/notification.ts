import { FindByIdInput, Notification, NotificationSearchInput, NotificationSettings, NotificationSettingsUpdateInput, Success, VisibilityType } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { defaultNotificationSettings, updateNotificationSettings } from "../../notify/notificationSettings";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";
import { parseJsonOrDefault } from "../../utils/objectTools";

export type EndpointsNotification = {
    Query: {
        notification: ApiEndpoint<FindByIdInput, FindOneResult<Notification>>;
        notifications: ApiEndpoint<NotificationSearchInput, FindManyResult<Notification>>;
        notificationSettings: ApiEndpoint<undefined, NotificationSettings>;
    },
    Mutation: {
        notificationMarkAsRead: ApiEndpoint<FindByIdInput, Success>;
        notificationMarkAllAsRead: ApiEndpoint<undefined, Success>;
        notificationSettingsUpdate: ApiEndpoint<NotificationSettingsUpdateInput, NotificationSettings>;
    }
}

const objectType = "Notification";
export const NotificationEndpoints: EndpointsNotification = {
    Query: {
        notification: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        notifications: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
        notificationSettings: async (_, __, { req }) => {
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const user = await prismaInstance.user.findUnique({
                where: { id },
                select: { notificationSettings: true },
            });
            if (!user) throw new CustomError("0402", "InternalError");
            return parseJsonOrDefault<NotificationSettings>(user.notificationSettings, defaultNotificationSettings);
        },
    },
    Mutation: {
        notificationMarkAsRead: async (_p, { input }, { req }) => {
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            const { count } = await prismaInstance.notification.updateMany({
                where: { AND: [{ user: { id: userId } }, { id: input.id }] },
                data: { isRead: true },
            });
            return { __typename: "Success", success: count > 0 };
        },
        notificationMarkAllAsRead: async (_, __, { req }) => {
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            await prismaInstance.notification.updateMany({
                where: { AND: [{ user: { id: userId } }, { isRead: false }] },
                data: { isRead: true },
            });
            return { __typename: "Success", success: true };
        },
        notificationSettingsUpdate: async (_, { input }, { req }) => {
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return updateNotificationSettings(input, id);
        },
    },
};
