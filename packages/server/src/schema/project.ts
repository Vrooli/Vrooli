import { gql } from 'apollo-server-express';
import { CODE, ProjectSortBy } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Project, ProjectInput, ProjectSearchInput, Success, ProjectCountInput } from './types';
import { Context } from '../context';
import { ProjectModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum ProjectSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input ProjectInput {
        id: ID
        name: String!
        description: String
        organizationId: ID
        resources: [ResourceInput!]
    }

    type Project {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        stars: Int!
        isStarred: Boolean
        score: Int!
        isUpvoted: Boolean
        resources: [Resource!]
        wallets: [Wallet!]
        creator: Contributor
        owner: Contributor
        starredBy: [User!]
        parent: Project
        forks: [Project!]!
        tags: [Tag!]!
        reports: [Report!]!
        comments: [Comment!]!
        routines: [Routine!]!
    }

    input ProjectSearchInput {
        userId: ID
        organizationId: ID
        parentId: ID
        reportId: ID
        ids: [ID!]
        sortBy: ProjectSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
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
        projectAdd(input: ProjectInput!): Project!
        projectUpdate(input: ProjectInput!): Project!
        projectDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    ProjectSortBy: ProjectSortBy,
    Query: {
        project: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            const data = await ProjectModel(prisma).findProject(req.userId ? req.userId : null, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<any> => {
            const data = await ProjectModel(prisma).searchProjects({}, req.userId ?? null, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        projectsCount: async (_parent: undefined, { input }: IWrap<ProjectCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return await ProjectModel(prisma).count({}, input);
        },
    },
    Mutation: {
        projectAdd: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await ProjectModel(prisma).addProject(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await ProjectModel(prisma).updateProject(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        projectDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await ProjectModel(prisma).deleteProject(req.userId, input);
        },
    }
}