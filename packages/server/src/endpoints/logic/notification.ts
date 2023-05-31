import { FindByIdInput, Notification, NotificationSearchInput, NotificationSettings, NotificationSettingsUpdateInput, Success, VisibilityType } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { parseNotificationSettings, updateNotificationSettings } from "../../notify";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../../types";

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
        notification: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        notifications: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
        notificationSettings: async (_, __, { prisma, req }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 250, req });
            const user = await prisma.user.findUnique({
                where: { id },
                select: { notificationSettings: true },
            });
            if (!user) throw new CustomError("0402", "InternalError", ["en"]);
            return parseNotificationSettings(user.notificationSettings);
        },
    },
    Mutation: {
        notificationMarkAsRead: async (_p, { input }, { prisma, req }, info) => {
            const { id: userId } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            const { count } = await prisma.notification.updateMany({
                where: { AND: [{ user: { id: userId } }, { id: input.id }] },
                data: { isRead: true },
            });
            return { __typename: "Success", success: count > 0 };
        },
        notificationMarkAllAsRead: async (_, __, { prisma, req }, info) => {
            const { id: userId } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            await prisma.notification.updateMany({
                where: { AND: [{ user: { id: userId } }, { isRead: false }] },
                data: { isRead: true },
            });
            return { __typename: "Success", success: true };
        },
        notificationSettingsUpdate: async (_, { input }, { prisma, req }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 100, req });
            return updateNotificationSettings(input, prisma, id);
        },
    },
};
