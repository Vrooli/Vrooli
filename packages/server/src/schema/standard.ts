import { gql } from 'apollo-server-express';
import { CODE, STANDARD_SORT_BY } from '@local/shared';
import { CustomError } from '../error';
import { StandardModel } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Standard, StandardInput, StandardSearchInput, Success } from './types';
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

    enum StandardSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
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
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        type: StandardType!
        schema: String!
        default: String
        isFile: Boolean!
        tags: [Tag!]!
        routineInputs: [Routine!]!
        routineOutputs: [Routine!]!
        starredBy: [User!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input StandardSearchInput {
        userId: Int
        ids: [ID!]
        sortBy: StandardSortBy
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type StandardSearchResult {
        pageInfo: PageInfo!
        edges: [StandardEdge!]!
    }

    # Return type for search result edge
    type StandardEdge {
        cursor: String!
        node: Standard!
    }

    extend type Query {
        standard(input: FindByIdInput!): Standard
        standards(input: StandardSearchInput!): StandardSearchResult!
        standardsCount: Count!
    }

    extend type Mutation {
        standardAdd(input: StandardInput!): Standard!
        standardUpdate(input: StandardInput!): Standard!
        standardDeleteMany(input: DeleteManyInput!): Count!
        standardReport(input: ReportInput!): Success!
    }
`

export const resolvers = {
    StandardType: StandardType,
    StandardSortBy: STANDARD_SORT_BY,
    Query: {
        standard: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard> | null> => {
            // Query database
            const dbModel = await StandardModel(prisma).findById(input, info);
            // Format data
            return dbModel ? StandardModel().toGraphQL(dbModel) : null;
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        standardsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        /**
         * Add a new standard
         * @returns Standard object if successful
         */
        standardAdd: async (_parent: undefined, { input }: IWrap<StandardInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            // Create object
            const dbModel = await StandardModel(prisma).create(input, info);
            // Format object to GraphQL type
            return StandardModel().toGraphQL(dbModel);
        },
        /**
         * Update standards you've created
         * @returns 
         */
        standardUpdate: async (_parent: undefined, { input }: IWrap<StandardInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Update object
            const dbModel = await StandardModel(prisma).update(input, info);
            // Format to GraphQL type
            return StandardModel().toGraphQL(dbModel);
        },
        /**
         * Delete standards you've created. Other standards must go through a reporting system
         * @returns 
         */
        standardDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
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
         standardReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await StandardModel(prisma).report(input);
            return { success };
        }
    }
}