import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input OrganizationInput {
        id: ID
        name: String!
        description: String
        resources: [ResourceInput!]
    }

    type Organization {
        id: ID!
        name: String!
        description: String
        resources: [Resource!]
        projects: [Project!]
        wallets: [Wallet!]
        starredBy: [User!]
        routines: [Routine!]
    }

    input OrganizationsQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        organization(input: FindByIdInput!): Organization
        organizations(input: OrganizationsQueryInput!): [Organization!]!
        organizationsCount: Count!
    }

    extend type Mutation {
        addOrganization(input: OrganizationInput!): Organization!
        updateOrganization(input: OrganizationInput!): Organization!
        deleteOrganization(input: DeleteOneInput): Boolean!
        reportOrganization(input: ReportInput!): Boolean!
    }
`

export const resolvers = {
    Query: {
        organization: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        organizations: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        organizationsCount: async (_parent: undefined, _args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addOrganization: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        updateOrganization: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        deleteOrganization: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        /**
         * Reports an organization. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportOrganization: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}