import { gql } from 'apollo-server-express';
import { CODE, PROJECT_SORT_BY } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteOneInput, FindByIdInput, Project, ProjectInput, ProjectSortBy, ProjectsQueryInput, ReportInput, Success } from './types';
import { Context } from '../context';
import { ProjectModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';
import { idArrayQuery } from '../prisma/fragments';
import { PrismaSelect } from '@paljs/plugins';

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

    input ProjectsQueryInput {
        userId: Int
        ids: [ID!]
        sortBy: ProjectSortBy
        searchString: String
        first: Int
        skip: Int
    }

    extend type Query {
        project(input: FindByIdInput!): Project
        projects(input: ProjectsQueryInput!): [Project!]!
        projectsCount: Count!
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
            return await ProjectModel(prisma).findById(input, info);
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectsQueryInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<any[] | null> => {
            // Create query for specified ids
            let idQuery;
            if (Array.isArray(input.ids)) idQuery = idArrayQuery(input.ids);
            // Create query for specified user
            let userQuery;
            if (input.userId) userQuery = { user: { id: input.userId } };
            // Determine sort order
            let sortQuery = ProjectModel().getSortQuery(input.sortBy ?? ProjectSortBy.DateUpdatedDesc);
            // Determine text search query
            let searchQuery;
            if (input.searchString) searchQuery = ProjectModel().getSearchStringQuery(input.searchString);
            // return query
            return await prisma.project.findMany({
                where: {
                    ...idQuery,
                    ...userQuery,
                    ...searchQuery
                },
                orderBy: sortQuery,
                skip: input.skip ?? 0,
                take: input.first ?? 20,
                ...(new PrismaSelect(info).value)
            })
            
        },
        projectsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        projectAdd: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await ProjectModel(prisma).create(input, info);
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be updating your own
            return await ProjectModel(prisma).update(input, info);
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