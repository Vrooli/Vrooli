import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, RunProjectScheduleSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RunProjectScheduleSortBy {
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
        WindowStartAsc
        WindowStartDesc
        WindowEndAsc
        WindowEndDesc
    }

    input RunProjectScheduleCreateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    input RunProjectScheduleUpdateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
    }
    type RunProjectSchedule {
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
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        wallets: [Wallet!]
    }

    type RunProjectSchedulePermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input RunProjectScheduleTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input RunProjectScheduleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type RunProjectScheduleTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input RunProjectScheduleSearchInput {
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

    type RunProjectScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type RunProjectScheduleEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        runProjectSchedule(input: FindByIdInput!): RunProjectSchedule
        runProjectSchedules(input: RunProjectScheduleSearchInput!): RunProjectScheduleSearchResult!
    }
`

const objectType = 'RunProjectSchedule';
export const resolvers: {
    RunProjectScheduleSortBy: typeof RunProjectScheduleSortBy;
    Query: {
        runProjectSchedule: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        runProjectSchedules: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
} = {
    RunProjectScheduleSortBy,
    Query: {
        runProjectSchedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        runProjectSchedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
}