import { Count, FindByIdInput, Notification, NotificationSearchInput, NotificationSettings, NotificationSettingsUpdateInput, NotificationSortBy, Success } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper, readOneHelper } from "../actions";
import { assertRequestFrom } from "../auth";
import { CustomError } from "../events";
import { rateLimit } from "../middleware";
import { parseNotificationSettings, updateNotificationSettings } from "../notify";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../types";

export const typeDef = gql`
    enum NotificationSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    type Notification {
        id: ID!
        created_at: Date!
        category: String!
        isRead: Boolean!
        title: String!
        description: String
        link: String
        imgLink: String
    }

    input NotificationSettingsCategoryUpdateInput {
        category: String!
        enabled: Boolean
        dailyLimit: Int
        toEmails: Boolean
        toSms: Boolean
        toPush: Boolean
    }

    input NotificationSettingsUpdateInput {
        includedEmails: [ID!]
        includedSms: [ID!]
        includedPush: [ID!]
        toEmails: Boolean
        toSms: Boolean
        toPush: Boolean
        dailyLimit: Int
        enabled: Boolean
        categories: [NotificationSettingsCategoryUpdateInput!]
    }

    type NotificationSettingsCategory {
        category: String!
        enabled: Boolean!
        dailyLimit: Int
        toEmails: Boolean
        toSms: Boolean
        toPush: Boolean
    }

    type NotificationSettings {
        includedEmails: [Email!]
        includedSms: [Phone!]
        includedPush: [PushDevice!]
        toEmails: Boolean
        toSms: Boolean
        toPush: Boolean
        dailyLimit: Int
        enabled: Boolean!
        categories: [NotificationSettingsCategory!]
    }

    input NotificationSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: NotificationSortBy
        take: Int
        visibility: VisibilityType
    }
    type NotificationSearchResult {
        pageInfo: PageInfo!
        edges: [NotificationEdge!]!
    }
    type NotificationEdge {
        cursor: String!
        node: Notification!
    }

    extend type Query {
        notification(input: FindByIdInput!): Notification
        notifications(input: NotificationSearchInput!): NotificationSearchResult!
        notificationSettings: NotificationSettings!
    }

    extend type Mutation {
        notificationMarkAsRead(input: FindByIdInput!): Success!
        notificationMarkAllAsRead: Count!
        notificationSettingsUpdate(input: NotificationSettingsUpdateInput!): NotificationSettings!
    }
`;

const objectType = "Notification";
export const resolvers: {
    NotificationSortBy: typeof NotificationSortBy;
    Query: {
        notification: GQLEndpoint<FindByIdInput, FindOneResult<Notification>>;
        notifications: GQLEndpoint<NotificationSearchInput, FindManyResult<Notification>>;
        notificationSettings: GQLEndpoint<undefined, NotificationSettings>;
    },
    Mutation: {
        notificationMarkAsRead: GQLEndpoint<FindByIdInput, Success>;
        notificationMarkAllAsRead: GQLEndpoint<undefined, Count>;
        notificationSettingsUpdate: GQLEndpoint<NotificationSettingsUpdateInput, NotificationSettings>;
    }
} = {
    NotificationSortBy,
    Query: {
        notification: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        notifications: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
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
        notificationMarkAsRead: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            throw new CustomError("0365", "NotImplemented", ["en"]);
        },
        notificationMarkAllAsRead: async (_, __, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            throw new CustomError("0366", "NotImplemented", ["en"]);
        },
        notificationSettingsUpdate: async (_, { input }, { prisma, req }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 100, req });
            return updateNotificationSettings(input, prisma, id);
        },
    },
};
