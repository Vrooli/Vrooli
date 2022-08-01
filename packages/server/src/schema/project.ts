import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { FindByIdOrHandleInput, Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, Success, ProjectCountInput, ProjectSearchResult, ProjectSortBy } from './types';
import { Context } from '../context';
import { countHelper, createHelper, ProjectModel, readManyHelper, readOneHelper, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

export const typeDef = gql`
    enum ProjectSortBy {
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input ProjectCreateInput {
        id: ID!
        createdByOrganizationId: ID
        createdByUserId: ID
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ProjectTranslationCreateInput!]
    }
    input ProjectUpdateInput {
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
    type Project {
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

    type ProjectPermission {
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canVote: Boolean!
    }

    input ProjectTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input ProjectTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ProjectTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input ProjectSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [BooleanSearchException!]
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
    }

    # Return type for search result
    type ProjectSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectEdge!]!
    }

    # Return type for search result edge
    type ProjectEdge {
        cursor: String!
        node: Project!
    }

    # Input for count
    input ProjectCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        project(input: FindByIdOrHandleInput!): Project
        projects(input: ProjectSearchInput!): ProjectSearchResult!
        projectsCount(input: ProjectCountInput!): Int!
    }

    extend type Mutation {
        projectCreate(input: ProjectCreateInput!): Project!
        projectUpdate(input: ProjectUpdateInput!): Project!
    }
`

export const resolvers = {
    ProjectSortBy: ProjectSortBy,
    Query: {
        project: async (_parent: undefined, { input }: IWrap<FindByIdOrHandleInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            await rateLimit({ info, max: 1000, req });
            return readOneHelper({ info, input, model: ProjectModel, prisma, userId: req.userId })
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectSearchInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<ProjectSearchResult> => {
            await rateLimit({ info, max: 1000, req });
            return readManyHelper({ info, input, model: ProjectModel, prisma, userId: req.userId })
        },
        projectsCount: async (_parent: undefined, { input }: IWrap<ProjectCountInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, max: 1000, req });
            return countHelper({ input, model: ProjectModel, prisma })
        },
    },
    Mutation: {
        projectCreate: async (_parent: undefined, { input }: IWrap<ProjectCreateInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            await rateLimit({ info, max: 100, byAccountOrKey: true, req });
            return createHelper({ info, input, model: ProjectModel, prisma, userId: req.userId })
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectUpdateInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            await rateLimit({ info, max: 250, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: ProjectModel, prisma, userId: req.userId })
        },
    }
}