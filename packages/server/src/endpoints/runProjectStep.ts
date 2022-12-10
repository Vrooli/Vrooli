import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, RunProjectStepSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RunProjectStepSortBy {
        ContextSwitchesAsc
        ContextSwitchesDesc
        OrderAsc
        OrderDesc
        TimeCompletedAsc
        TimeCompletedDesc
        TimeStartedAsc
        TimeStartedDesc
        TimeElapsedAsc
        TimeElapsedDesc
    }

    input RunProjectStepCreateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        rootId: ID!
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    input RunProjectStepUpdateInput {
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
    }
    type RunProjectStep {
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

    type RunProjectStepPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input RunProjectStepTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input RunProjectStepTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type RunProjectStepTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input RunProjectStepSearchInput {
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

    type RunProjectStepSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type RunProjectStepEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        runProjectStep(input: FindByIdInput!): RunProjectStep
        runProjectSteps(input: RunProjectStepSearchInput!): RunProjectStepSearchResult!
    }

    extend type Mutation {
        runProjectStepCreate(input: RunProjectStepCreateInput!): RunProjectStep!
        runProjectStepUpdate(input: RunProjectStepUpdateInput!): RunProjectStep!
    }
`

const objectType = 'RunProjectStep';
export const resolvers: {
    RunProjectStepSortBy: typeof RunProjectStepSortBy;
    Query: {
        runProjectStep: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        runProjectSteps: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        runProjectStepCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        runProjectStepUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    RunProjectStepSortBy,
    Query: {
        runProjectStep: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        runProjectSteps: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        runProjectStepCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        runProjectStepUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}