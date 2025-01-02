import { FindByIdInput, Notification, NotificationSearchInput, NotificationSettings, NotificationSettingsUpdateInput, Success, VisibilityType } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { defaultNotificationSettings, updateNotificationSettings } from "../../notify/notificationSettings";
import { ApiEndpoint, FindManyResult, FindOneResult } from "../../types";
import { parseJsonOrDefault } from "../../utils/objectTools";

export type EndpointsNotification = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Notification>>;
    findMany: ApiEndpoint<NotificationSearchInput, FindManyResult<Notification>>;
    getSettings: ApiEndpoint<undefined, NotificationSettings>;
    markAsRead: ApiEndpoint<FindByIdInput, Success>;
    markAllAsRead: ApiEndpoint<undefined, Success>;
    updateSettings: ApiEndpoint<NotificationSettingsUpdateInput, NotificationSettings>;
}

const objectType = "Notification";
export const notification: EndpointsNotification = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    getSettings: async (_, __, { req }) => {
        const { id } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const user = await prismaInstance.user.findUnique({
            where: { id },
            select: { notificationSettings: true },
        });
        if (!user) throw new CustomError("0402", "InternalError");
        return parseJsonOrDefault<NotificationSettings>(user.notificationSettings, defaultNotificationSettings);
    },
    markAsRead: async (_p, { input }, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        const { count } = await prismaInstance.notification.updateMany({
            where: { AND: [{ user: { id: userId } }, { id: input.id }] },
            data: { isRead: true },
        });
        return { __typename: "Success", success: count > 0 };
    },
    markAllAsRead: async (_, __, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        await prismaInstance.notification.updateMany({
            where: { AND: [{ user: { id: userId } }, { isRead: false }] },
            data: { isRead: true },
        });
        return { __typename: "Success", success: true };
    },
    updateSettings: async (_, { input }, { req }) => {
        const { id } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return updateNotificationSettings(input, id);
    },
};
