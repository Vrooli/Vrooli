import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { ResourceModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Resource, ResourceInput, ResourcesQueryInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum ResourceFor {
        ORGANIZATION
        PROJECT
        ROUTINE_CONTEXTUAL
        ROUTINE_EXTERNAL
        ROUTINE_DONATION
        USER
    }

    input ResourceInput {
        id: ID
        name: String!
        description: String
        link: String!
        displayUrl: String
        createdFor: ResourceFor!
        forId: ID!
    }

    type Resource {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        link: String!
        displayUrl: String
        organization_resources: [Organization!]!
        project_resources: [Project!]!
        routine_resources_contextual: [Routine!]!
        routine_resources_external: [Routine!]!
        routine_resources_donation: [Routine!]!
        user_resources: [User!]!
        starredBy: [User!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input ResourcesQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        resource(input: FindByIdInput!): Resource
        resources(input: ResourcesQueryInput!): [Resource!]!
        resourcesCount: Count!
    }

    extend type Mutation {
        resourceAdd(input: ResourceInput!): Resource!
        resourceUpdate(input: ResourceInput!): Resource!
        resourceDeleteMany(input: DeleteManyInput!): Count!
        resourceReport(input: ReportInput!): Success!
    }
`

export const resolvers = {
    Query: {
        resource: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource> | null> => {
            return await ResourceModel(prisma).findById(input, info);
        },
        resources: async (_parent: undefined, { input }: IWrap<ResourcesQueryInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        resourcesCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        resourceAdd: async (_parent: undefined, { input }: IWrap<ResourceInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).create(input, info)
        },
        resourceUpdate: async (_parent: undefined, { input }: IWrap<ResourceInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).update(input, info);
        },
        resourceDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).deleteMany(input);
        },
        /**
         * Reports a resource. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         resourceReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: any): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await ResourceModel(prisma).report(input);
            return { success };
        }
    }
}