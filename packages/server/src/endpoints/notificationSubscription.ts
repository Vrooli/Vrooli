import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, NotificationSubscriptionSortBy, SubscribableObject, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionUpdateInput, NotificationSubscriptionSearchInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum NotificationSubscriptionSortBy {
        DateCreatedAsc
        DateCreatedDesc
        ObjectTypeAsc
        ObjectTypeDesc
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
        SmartContract
        Standard
    }

    union SubscribedObject = Api | Comment | Issue | Meeting | Note | Organization | Project | PullRequest | Question | Quiz | Report | Routine | SmartContract | Standard

    input NotificationSubscriptionCreateInput {
        id: ID!
        objectType: SubscribableObject!
        objectConnect: ID!
        silent: Boolean
    }
    input NotificationSubscriptionUpdateInput {
        id: ID!
        silent: Boolean
    }
    type NotificationSubscription {
        id: ID!
        created_at: Date!
        object: SubscribedObject!
        silent: Boolean!
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