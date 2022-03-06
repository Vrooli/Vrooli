import { gql } from 'apollo-server-express';
import { ProjectSortBy } from '@local/shared';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, Success, ProjectCountInput, ProjectSearchResult } from './types';
import { Context } from '../context';
import { countHelper, createHelper, deleteOneHelper, ProjectModel, readManyHelper, readOneHelper, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum ProjectSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
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
        createdByOrganizationId: ID
        createdByUserId: ID
        description: String
        isComplete: Boolean
        name: String!
        parentId: ID
        resourcesCreate: [ResourceCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    input ProjectUpdateInput {
        id: ID!
        description: String
        isComplete: Boolean
        name: String
        organizationId: ID
        parentId: ID
        userId: ID
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    type Project {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        description: String
        isComplete: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        name: String!
        role: MemberRole
        score: Int!
        stars: Int!
        comments: [Comment!]!
        creator: Contributor
        forks: [Project!]!
        owner: Contributor
        parent: Project
        reports: [Report!]!
        resources: [Resource!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        wallets: [Wallet!]
    }

    input ProjectSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        minScore: Int
        minStars: Int
        organizationId: ID
        parentId: ID
        reportId: ID
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
        project(input: FindByIdInput!): Project
        projects(input: ProjectSearchInput!): ProjectSearchResult!
        projectsCount(input: ProjectCountInput!): Int!
    }

    extend type Mutation {
        projectCreate(input: ProjectCreateInput!): Project!
        projectUpdate(input: ProjectUpdateInput!): Project!
        projectDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    ProjectSortBy: ProjectSortBy,
    Query: {
        project: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            return readOneHelper(req.userId, input, info, ProjectModel(prisma));
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ProjectSearchResult> => {
            return readManyHelper(req.userId, input, info, ProjectModel(prisma));
        },
        projectsCount: async (_parent: undefined, { input }: IWrap<ProjectCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return countHelper(input, ProjectModel(prisma));
        },
    },
    Mutation: {
        projectCreate: async (_parent: undefined, { input }: IWrap<ProjectCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            return createHelper(req.userId, input, info, ProjectModel(prisma));
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            return updateHelper(req.userId, input, info, ProjectModel(prisma));
        },
        projectDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, ProjectModel(prisma));
        },
    }
}