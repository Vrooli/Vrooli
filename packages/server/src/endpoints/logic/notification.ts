import { FindByIdInput, Notification, NotificationSearchInput, NotificationSettings, NotificationSettingsUpdateInput, Success, VisibilityType } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { assertRequestFrom } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { defaultNotificationSettings, updateNotificationSettings } from "../../notify/notificationSettings";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../../types";
import { parseJsonOrDefault } from "../../utils/objectTools";

export type EndpointsNotification = {
    Query: {
        notification: GQLEndpoint<FindByIdInput, FindOneResult<Notification>>;
        notifications: GQLEndpoint<NotificationSearchInput, FindManyResult<Notification>>;
        notificationSettings: GQLEndpoint<undefined, NotificationSettings>;
    },
    Mutation: {
        notificationMarkAsRead: GQLEndpoint<FindByIdInput, Success>;
        notificationMarkAllAsRead: GQLEndpoint<undefined, Success>;
        notificationSettingsUpdate: GQLEndpoint<NotificationSettingsUpdateInput, NotificationSettings>;
    }
}

const objectType = "Notification";
export const NotificationEndpoints: EndpointsNotification = {
    Query: {
        notification: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        notifications: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
        notificationSettings: async (_, __, { req }) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 250, req });
            const user = await prismaInstance.user.findUnique({
                where: { id },
                select: { notificationSettings: true },
            });
            if (!user) throw new CustomError("0402", "InternalError", ["en"]);
            return parseJsonOrDefault<NotificationSettings>(user.notificationSettings, defaultNotificationSettings);
        },
    },
    Mutation: {
        notificationMarkAsRead: async (_p, { input }, { req }) => {
            const { id: userId } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
            const { count } = await prismaInstance.notification.updateMany({
                where: { AND: [{ user: { id: userId } }, { id: input.id }] },
                data: { isRead: true },
            });
            return { __typename: "Success", success: count > 0 };
        },
        notificationMarkAllAsRead: async (_, __, { req }) => {
            const { id: userId } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
            await prismaInstance.notification.updateMany({
                where: { AND: [{ user: { id: userId } }, { isRead: false }] },
                data: { isRead: true },
            });
            return { __typename: "Success", success: true };
        },
        notificationSettingsUpdate: async (_, { input }, { req }) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 100, req });
            return updateNotificationSettings(input, id);
        },
    },
};
