import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Organization, OrganizationInput, OrganizationsQueryInput, ReportInput } from './types';
import { Context } from '../context';
import { OrganizationModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';

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
        comments: [Comment!]!
        resources: [Resource!]!
        wallets: [Wallet!]!
        projects: [Project!]!
        wallets: [Wallet!]!
        starredBy: [User!]!
        routines: [Routine!]!
        tags: [Tag!]!
        reports: [Report!]!
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
        organization: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization> | null> => {
            return await OrganizationModel(prisma).findById(input, info);
        },
        organizations: async (_parent: undefined, { input }: IWrap<OrganizationsQueryInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        organizationsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addOrganization: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await OrganizationModel(prisma).create(input, info);
        },
        updateOrganization: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be updating your own
            return await OrganizationModel(prisma).update(input, info);
        },
        deleteOrganization: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be deleting your own
            return await OrganizationModel(prisma).delete(input);
        },
        /**
         * Reports an organization. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportOrganization: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await OrganizationModel(prisma).report(input);
        }
    }
}