import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteOneInput, FindByIdInput, Organization, OrganizationInput, OrganizationsQueryInput, ReportInput, Success } from './types';
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
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        comments: [Comment!]!
        resources: [Resource!]!
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
        organizationAdd(input: OrganizationInput!): Organization!
        organizationUpdate(input: OrganizationInput!): Organization!
        organizationDeleteOne(input: DeleteOneInput): Success!
        organizationReport(input: ReportInput!): Success!
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
        organizationsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        organizationAdd: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await OrganizationModel(prisma).create(input, info);
        },
        organizationUpdate: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be updating your own
            return await OrganizationModel(prisma).update(input, info);
        },
        organizationDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be deleting your own
            const success = await OrganizationModel(prisma).delete(input);
            return { success };
        },
        /**
         * Reports an organization. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
        organizationReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await OrganizationModel(prisma).report(input);
            return { success };
        }
    }
}