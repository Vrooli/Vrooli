import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, UserScheduleSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum UserScheduleSortBy {
        TitleAsc
        TitleDesc
        EventStartAsc
        EventStartDesc
        EventEndAsc
        EventEndDesc
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
    }

    input UserScheduleCreateInput {
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
    input UserScheduleUpdateInput {
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
    type UserSchedule {
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
        createdBy: User
        forks: [Project!]!
        owner: Owner
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

    type UserSchedulePermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input UserScheduleTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input UserScheduleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type UserScheduleTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input UserScheduleSearchInput {
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

    type UserScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type UserScheduleEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        userSchedule(input: FindByIdInput!): UserSchedule
        userSchedules(input: UserScheduleSearchInput!): UserScheduleSearchResult!
    }

    extend type Mutation {
        userScheduleCreate(input: UserScheduleCreateInput!): UserSchedule!
        userScheduleUpdate(input: UserScheduleUpdateInput!): UserSchedule!
    }
`

const objectType = 'UserSchedule';
export const resolvers: {
    UserScheduleSortBy: typeof UserScheduleSortBy;
    Query: {
        userSchedule: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        userSchedules: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        userScheduleCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        userScheduleUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    UserScheduleSortBy,
    Query: {
        userSchedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        userSchedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        userScheduleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        userScheduleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}