import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Project, ProjectInput, ProjectsQueryInput, ReportInput } from './types';
import { Context } from '../context';
import { ProjectModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
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
        name: String!
        description: String
        resources: [Resource!]
        wallets: [Wallet!]
        users: [User!]
        organizations: [Organization!]
        starredBy: [User!]
    }

    input ProjectsQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        project(input: FindByIdInput!): Project
        projects(input: ProjectsQueryInput!): [Project!]!
        projectsCount: Int!
    }

    extend type Mutation {
        addProject(input: ProjectInput!): Project!
        updateProject(input: ProjectInput!): Project!
        deleteProject(input: DeleteOneInput!): Boolean!
        reportProject(input: ReportInput!): Boolean!
    }
`

export const resolvers = {
    Query: {
        project: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            return await ProjectModel(prisma).findById(input, info);
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectsQueryInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        projectsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addProject: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await ProjectModel(prisma).create(input, info);
        },
        updateProject: async (_parent: undefined, { input }: IWrap<ProjectInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await ProjectModel(prisma).update(input, info);
        },
        deleteProject: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await ProjectModel(prisma).delete(input);
        },
        /**
         * Reports a project. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportProject: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ProjectModel(prisma).report(input);
        }
    }
}