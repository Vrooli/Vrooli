import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

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
        project: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        projects: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        projectsCount: async (_parent: undefined, _args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addProject: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        updateProject: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        deleteProject: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        /**
         * Reports a project. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportProject: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}