import { NotificationSortBy } from "@local/shared";
import { EndpointsNotification, NotificationEndpoints } from "../logic/notification";

export const typeDef = `#graphql
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
        userId: ID
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
        notificationMarkAllAsRead: Success!
        notificationSettingsUpdate(input: NotificationSettingsUpdateInput!): NotificationSettings!
    }
`;

export const resolvers: {
    NotificationSortBy: typeof NotificationSortBy;
    Query: EndpointsNotification["Query"];
    Mutation: EndpointsNotification["Mutation"];
} = {
    NotificationSortBy,
    ...NotificationEndpoints,
};
