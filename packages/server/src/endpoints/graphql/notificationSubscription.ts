import { NotificationSubscriptionSortBy, SubscribableObject } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsNotificationSubscription, NotificationSubscriptionEndpoints } from "../logic/notificationSubscription";

export const typeDef = gql`
    enum NotificationSubscriptionSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    enum SubscribableObject {
        Api
        Comment
        Issue
        Meeting
        Note
        Organization
        Project
        PullRequest
        Question
        Quiz
        Report
        Routine
        Schedule
        SmartContract
        Standard
    }

    union SubscribedObject = Api | Comment | Issue | Meeting | Note | Organization | Project | PullRequest | Question | Quiz | Report | Routine | Schedule | SmartContract | Standard

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
