import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { StandardModel } from '../models';
import pkg from '@prisma/client';
const { StandardType } = pkg;

export const typeDef = gql`
    enum StandardType {
        STRING
        NUMBER
        BOOLEAN
        OBJECT
        ARRAY
        FILE
        URL
    }

    input StandardInput {
        id: ID
        name: String
        description: String
        type: StandardType
        schema: String
        default: String
        isFile: Boolean
        tags: [TagInput!]
    }

    type Standard {
        id: ID!
        name: String!
        description: String
        type: StandardType!
        schema: String!
        default: String
        isFile: Boolean!
        tags: [Tag!]!
    }

    input StandardsQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        standard(input: FindByIdInput!): Standard
        standards(input: StandardsQueryInput!): [Standard!]!
        standardsCount: Int!
    }

    extend type Mutation {
        addStandard(input: ResourceInput!): Resource!
        updateStandard(input: ResourceInput!): Resource!
        deleteStandards(input: DeleteManyInput!): Count!
        reportStandard(input: ReportInput!): Boolean!
    }
`

export const resolvers = {
    StandardType: StandardType,
    Query: {
        standard: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        standards: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        standardsCount: async (_parent: undefined, _args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        /**
         * Add a new standard
         * @returns Standard object if successful
         */
        addStandard: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Create object
            return await new StandardModel(context.prisma).create(input, info)
        },
        /**
         * Update standards you've created
         * @returns 
         */
        updateStandard: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Update object
            return await new StandardModel(context.prisma).update(input, info);
        },
        /**
         * Delete standards you've created. Other standards must go through a reporting system
         * @returns 
         */
        deleteStandards: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Delete objects
            return await new StandardModel(context.prisma).deleteMany(input.ids);
        },
        /**
         * Reports a standard. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportStandard: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}