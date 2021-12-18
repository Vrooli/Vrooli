import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { StandardModel } from '../models';
import { IWrap } from '../types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Standard, StandardInput, StandardsQueryInput } from './types';
import { Context } from '../context';
import pkg from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
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
        addStandard(input: StandardInput!): Standard!
        updateStandard(input: StandardInput!): Standard!
        deleteStandards(input: DeleteManyInput!): Count!
        reportStandard(input: ReportInput!): Boolean!
    }
`

export const resolvers = {
    StandardType: StandardType,
    Query: {
        standard: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<Standard | null> => {
            return await StandardModel(prisma).findById(input, info);
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardsQueryInput>, context: Context, info: GraphQLResolveInfo): Promise<Standard[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        standardsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        /**
         * Add a new standard
         * @returns Standard object if successful
         */
        addStandard: async (_parent: undefined, { input }: IWrap<StandardInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Standard> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            return await StandardModel(prisma).create(input, info)
        },
        /**
         * Update standards you've created
         * @returns 
         */
        updateStandard: async (_parent: undefined, { input }: IWrap<StandardInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Standard> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await StandardModel(prisma).update(input, info);
        },
        /**
         * Delete standards you've created. Other standards must go through a reporting system
         * @returns 
         */
        deleteStandards: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            return await StandardModel(prisma).deleteMany(input);
        },
        /**
         * Reports a standard. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportStandard: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await StandardModel(prisma).report(input);
        }
    }
}