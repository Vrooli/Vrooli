import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, RunRoutineScheduleSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RunRoutineScheduleSortBy {
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
        WindowStartAsc
        WindowStartDesc
        WindowEndAsc
        WindowEndDesc
    }

    input RunRoutineScheduleCreateInput {
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
    input RunRoutineScheduleUpdateInput {
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
    type RunRoutineSchedule {
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

    type RunRoutineSchedulePermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input RunRoutineScheduleTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input RunRoutineScheduleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type RunRoutineScheduleTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input RunRoutineScheduleSearchInput {
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

    type RunRoutineScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type RunRoutineScheduleEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        runRoutineSchedule(input: FindByIdInput!): RunRoutineSchedule
        runRoutineSchedules(input: RunRoutineScheduleSearchInput!): RunRoutineScheduleSearchResult!
    }
`

const objectType = 'RunRoutineSchedule';
export const resolvers: {
    RunRoutineScheduleSortBy: typeof RunRoutineScheduleSortBy;
    Query: {
        runRoutineSchedule: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        runRoutineSchedules: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
} = {
    RunRoutineScheduleSortBy,
    Query: {
        runRoutineSchedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        runRoutineSchedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
}