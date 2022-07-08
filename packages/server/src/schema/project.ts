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
        isComplete: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ProjectTranslationCreateInput!]
    }
    input ProjectUpdateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        organizationId: ID
        parentId: ID
        userId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
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
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        role: MemberRole
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        creator: Contributor
        forks: [Project!]!
        owner: Contributor
        parent: Project
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        translations: [ProjectTranslation!]!
        wallets: [Wallet!]
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
        project: async (_parent: undefined, { input }: IWrap<FindByIdOrHandleInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, ProjectModel(context.prisma));
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<ProjectSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, ProjectModel(context.prisma));
        },
        projectsCount: async (_parent: undefined, { input }: IWrap<ProjectCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, ProjectModel(context.prisma));
        },
    },
    Mutation: {
        projectCreate: async (_parent: undefined, { input }: IWrap<ProjectCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            await rateLimit({ context, info, max: 100, byAccount: true });
            return createHelper(context.req.userId, input, info, ProjectModel(context.prisma));
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            await rateLimit({ context, info, max: 250, byAccount: true });
            return updateHelper(context.req.userId, input, info, ProjectModel(context.prisma));
        },
    }
}