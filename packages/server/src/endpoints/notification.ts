import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, NotificationSortBy, Notification, NotificationSearchInput, Success, Count } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { CustomError } from '../events';

export const typeDef = gql`
    enum NotificationSortBy {
        CategoryAsc
        CategoryDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        TitleAsc
        TitleDesc
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
    }

    extend type Mutation {
        notificationMarkAsRead(input: FindByIdInput!): Success!
        notificationMarkAllAsRead: Count!
    }
`

const objectType = 'Notification';
export const resolvers: {
    NotificationSortBy: typeof NotificationSortBy;
    Query: {
        notification: GQLEndpoint<FindByIdInput, FindOneResult<Notification>>;
        notifications: GQLEndpoint<NotificationSearchInput, FindManyResult<Notification>>;
    },
    Mutation: {
        notificationMarkAsRead: GQLEndpoint<FindByIdInput, Success>;
        notificationMarkAllAsRead: GQLEndpoint<undefined, Count>;
    }
} = {
    NotificationSortBy,
    Query: {
        notification: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        notifications: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        notificationMarkAsRead: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            throw new CustomError('0365', 'NotImplemented', ['en'])
        },
        notificationMarkAllAsRead: async (_, __, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            throw new CustomError('0366', 'NotImplemented', ['en'])
        }
    },
}