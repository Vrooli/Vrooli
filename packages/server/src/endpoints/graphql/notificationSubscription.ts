import { NotificationSubscriptionSortBy, SubscribableObject } from "@local/shared";
import { EndpointsNotificationSubscription, NotificationSubscriptionEndpoints } from "../logic/notificationSubscription";

export const typeDef = `#graphql
    enum NotificationSubscriptionSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    enum SubscribableObject {
        Api
        Comment
        Code
        Issue
        Meeting
        Note
        Project
        PullRequest
        Question
        Quiz
        Report
        Routine
        Schedule
        Standard
        Team
    }

    union SubscribedObject = Api | Code | Comment | Issue | Meeting | Note | Project | PullRequest | Question | Quiz | Report | Routine | Schedule | Standard | Team

    input NotificationSubscriptionCreateInput {
        id: ID!
        context: String
        objectType: SubscribableObject!
        silent: Boolean
        objectConnect: ID!
    }
    input NotificationSubscriptionUpdateInput {
        id: ID!
        context: String
        silent: Boolean
    }
    type NotificationSubscription {
        id: ID!
        created_at: Date!
        context: String
        silent: Boolean!
        object: SubscribedObject!
    }

    input NotificationSubscriptionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        silent: Boolean
        objectType: SubscribableObject
        objectId: ID
        sortBy: NotificationSubscriptionSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type NotificationSubscriptionSearchResult {
        pageInfo: PageInfo!
        edges: [NotificationSubscriptionEdge!]!
    }

    type NotificationSubscriptionEdge {
        cursor: String!
        node: NotificationSubscription!
    }

    extend type Query {
        notificationSubscription(input: FindByIdInput!): NotificationSubscription
        notificationSubscriptions(input: NotificationSubscriptionSearchInput!): NotificationSubscriptionSearchResult!
    }

    extend type Mutation {
        notificationSubscriptionCreate(input: NotificationSubscriptionCreateInput!): NotificationSubscription!
        notificationSubscriptionUpdate(input: NotificationSubscriptionUpdateInput!): NotificationSubscription!
    }
`;

export const resolvers: {
    NotificationSubscriptionSortBy: typeof NotificationSubscriptionSortBy;
    SubscribableObject: typeof SubscribableObject;
    Query: EndpointsNotificationSubscription["Query"];
    Mutation: EndpointsNotificationSubscription["Mutation"];
} = {
    NotificationSubscriptionSortBy,
    SubscribableObject,
    ...NotificationSubscriptionEndpoints,
};
