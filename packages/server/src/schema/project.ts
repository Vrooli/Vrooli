import { gql } from 'apollo-server-express';
import { CODE, PROJECT_SORT_BY } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteOneInput, FindByIdInput, Project, ProjectInput, ProjectSortBy, ProjectSearchInput, ReportInput, Success, ProjectSearchResult, ProjectCountInput } from './types';
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
        organizations: [OrganizationInput!]
        users: [UserInput!]
        resources: [ResourceInput!]
    }

    type Project {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        resources: [Resource!]
        wallets: [Wallet!]
        users: [User!]
        organizations: [Organization!]
        starredBy: [User!]
        parent: Project
        forks: [Project!]!
        tags: [Tag!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input ProjectSearchInput {
        userId: ID
        ids: [ID!]
        sortBy: ProjectSortBy
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
        createdMetric: MetricTimeFrame
        updatedMetric: MetricTimeFrame
    }

    extend type Query {
        project(input: FindByIdInput!): Project
        projects(input: ProjectSearchInput!): ProjectSearchResult!
        projectsCount(input: ProjectCountInput!): number!
    }

    extend type Mutation {
        projectAdd(input: ProjectInput!): Project!
        projectUpdate(input: ProjectInput!): Project!
        projectDeleteOne(input: DeleteOneInput!): Success!
        projectReport(input: ReportInput!): Success!
    }
`

export const resolvers = {
    ProjectSortBy: PROJECT_SORT_BY,
    Query: {
        project: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            // Query database
            const dbModel = await ProjectModel(prisma).findById(input, info);
            // Format data
            return dbModel ? ProjectModel().toGraphQL(dbModel) : null;
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectSearchInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<any> => {
            // Create query for specified user
            const userQuery = input.userId ? { user: { id: input.userId } } : undefined;
            // return search query
            return await ProjectModel(prisma).search({...userQuery,}, input, info);
        },
        projectsCount: async (_parent: undefined, { input }: IWrap<ProjectCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await ProjectModel(prisma).count({}, input);
        },
    },
    Mutation: {
        projectAdd: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            // Create object
            const dbModel = await ProjectModel(prisma).create(input, info);
            // Format object to GraphQL type
            return ProjectModel().toGraphQL(dbModel);
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be updating your own
            // Update object
            const dbModel = await ProjectModel(prisma).update(input, info);
            // Format to GraphQL type
            return ProjectModel().toGraphQL(dbModel);
        },
        projectDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            const success = await ProjectModel(prisma).delete(input);
            return { success };
        },
        /**
         * Reports a project. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         projectReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await ProjectModel(prisma).report(input);
            return { success };
        }
    }
}