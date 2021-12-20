import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { ResourceModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Resource, ResourceInput, ResourcesQueryInput } from './types';
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
        name: String!
        description: String
        link: String!
        displayUrl: String
    }

    input ResourcesQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        resource(input: FindByIdInput!): Resource
        resources(input: ResourcesQueryInput!): [Resource!]!
        resourcesCount: Int!
    }

    extend type Mutation {
        addResource(input: ResourceInput!): Resource!
        updateResource(input: ResourceInput!): Resource!
        deleteResources(input: DeleteManyInput!): Count!
        reportResource(input: ReportInput!): Boolean!
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
        resourcesCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addResource: async (_parent: undefined, { input }: IWrap<ResourceInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).create(input, info)
        },
        updateResource: async (_parent: undefined, { input }: IWrap<ResourceInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).update(input, info);
        },
        deleteResources: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).deleteMany(input);
        },
        /**
         * Reports a resource. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportResource: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: any): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).report(input);
        }
    }
}