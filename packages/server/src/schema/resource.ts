import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { ResourceModel } from '../models';

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

    extend type Query {
        resource(id: ID!): Resource
        resources(first: Int, skip: Int): [Resource!]!
        resourcesCount: Int!
    }

    extend type Mutation {
        addResource(input: ResourceInput!): Resource!
        updateResource(input: ResourceInput!): Resource!
        deleteResources(ids: [ID!]!): Count!
        reportResource(id: ID!): Boolean!
    }
`

export const resolvers = {
    Query: {
        resource: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        resources: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        resourcesCount: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addResource: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Create resource object
            return await new ResourceModel(context.prisma).create(args.input, info)
        },
        updateResource: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Update resource object
            return await new ResourceModel(context.prisma).update(args.input, info);
        },
        deleteResources: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Delete resource objects
            return await new ResourceModel(context.prisma).deleteMany(args.ids);
        },
        /**
         * Reports a resource. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportResource: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}