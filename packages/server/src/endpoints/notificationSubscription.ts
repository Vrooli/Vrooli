import { FindByIdInput, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionSortBy, NotificationSubscriptionUpdateInput, SubscribableObject } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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
`

const objectType = 'NotificationSubscription';
export const resolvers: {
    NotificationSubscriptionSortBy: typeof NotificationSubscriptionSortBy;
    SubscribableObject: typeof SubscribableObject;
    Query: {
        notificationSubscription: GQLEndpoint<FindByIdInput, FindOneResult<NotificationSubscription>>;
        notificationSubscriptions: GQLEndpoint<NotificationSubscriptionSearchInput, FindManyResult<NotificationSubscription>>;
    },
    Mutation: {
        notificationSubscriptionCreate: GQLEndpoint<NotificationSubscriptionCreateInput, CreateOneResult<NotificationSubscription>>;
        notificationSubscriptionUpdate: GQLEndpoint<NotificationSubscriptionUpdateInput, UpdateOneResult<NotificationSubscription>>;
    }
} = {
    NotificationSubscriptionSortBy,
    SubscribableObject,
    Query: {
        notificationSubscription: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        notificationSubscriptions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        notificationSubscriptionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        notificationSubscriptionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}