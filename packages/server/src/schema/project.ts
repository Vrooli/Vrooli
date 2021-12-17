import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { IWrap } from 'types';
import { DeleteOneInput, FindByIdInput, Project, ProjectInput, ProjectsQueryInput, ReportInput } from './types';

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
        project: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: any, info: any): Promise<Project> => {
            throw new CustomError(CODE.NotImplemented);
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectsQueryInput>, context: any, info: any): Promise<Project[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        projectsCount: async (_parent: undefined, _args: undefined, context: any, info: any): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addProject: async (_parent: undefined, { input }: IWrap<ProjectInput>, context: any, info: any): Promise<Project> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        updateProject: async (_parent: undefined, { input }: IWrap<ProjectInput>, context: any, info: any): Promise<Project> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        deleteProject: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, context: any, _info: any): Promise<boolean> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        /**
         * Reports a project. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportProject: async (_parent: undefined, { input }: IWrap<ReportInput>, context: any, _info: any): Promise<boolean> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}