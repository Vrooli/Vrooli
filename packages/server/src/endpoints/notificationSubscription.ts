import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, NotificationSubscriptionSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum NotificationSubscriptionSortBy {
        ObjectTypeAsc
        ObjectTypeDesc
    }

    input NotificationSubscriptionCreateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        rootId: ID!
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ProjectTranslationCreateInput!]
    }
    input NotificationSubscriptionUpdateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        organizationId: ID
        userId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [ProjectTranslationCreateInput!]
        translationsUpdate: [ProjectTranslationUpdateInput!]
    }
    type NotificationSubscription {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        handle: String
        isComplete: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        creator: Contributor
        forks: [Project!]!
        owner: Contributor
        parent: Project
        permissionsProject: ProjectPermission!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        translations: [ProjectTranslation!]!
        wallets: [Wallet!]
    }

    type NotificationSubscriptionPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input NotificationSubscriptionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input NotificationSubscriptionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type NotificationSubscriptionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input NotificationSubscriptionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        organizationId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type NotificationSubscriptionSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type NotificationSubscriptionEdge {
        cursor: String!
        node: Api!
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
    Query: {
        notificationSubscription: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        notificationSubscriptions: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        notificationSubscriptionCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        notificationSubscriptionUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    NotificationSubscriptionSortBy,
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