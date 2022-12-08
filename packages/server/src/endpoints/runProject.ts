import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, RunProjectSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RunProjectSortBy {
        ContextSwitchesAsc
        ContextSwitchesDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateStartedAsc
        DateStartedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        RunRoutinesAsc
        RunRoutinesDesc
        StepsAsc
        StepsDesc
    }

    input RunProjectCreateInput {
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
    input RunProjectUpdateInput {
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
    type RunProject {
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

    type RunProjectPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input RunProjectTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input RunProjectTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type RunProjectTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input RunProjectSearchInput {
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

    type RunProjectSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type RunProjectEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        runProject(input: FindByIdInput!): RunProject
        runProjects(input: RunProjectSearchInput!): RunProjectSearchResult!
    }

    extend type Mutation {
        runProjectCreate(input: RunProjectCreateInput!): RunProject!
        runProjectUpdate(input: RunProjectUpdateInput!): RunProject!
    }
`

const objectType = 'RunProject';
export const resolvers: {
    RunProjectSortBy: typeof RunProjectSortBy;
    Query: {
        runProject: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        runProjects: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        runProjectCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        runProjectUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    RunProjectSortBy,
    Query: {
        runProject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        runProjects: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        runProjectCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        runProjectUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}