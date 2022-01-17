import { gql } from 'apollo-server-express';
import { CODE, StandardSortBy } from '@local/shared';
import { CustomError } from '../error';
import { StandardModel } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, Standard, StandardCountInput, StandardInput, StandardSearchInput, Success } from './types';
import { Context } from '../context';
import pkg from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
const { StandardType } = pkg;

export const typeDef = gql`
    enum StandardType {
        String
        Number
        Boolean
        Object
        Array
        File
        Url
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
        stars: Int!
        votes: Int!
        isUpvoted: Boolean!
        creator: Contributor
        tags: [Tag!]!
        routineInputs: [Routine!]!
        routineOutputs: [Routine!]!
        starredBy: [User!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input StandardSearchInput {
        userId: ID
        organizationId: ID
        routineId: ID
        reportId: ID
        ids: [ID!]
        sortBy: StandardSortBy
        searchString: String
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
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

    # Input for count
    input StandardCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        standard(input: FindByIdInput!): Standard
        standards(input: StandardSearchInput!): StandardSearchResult!
        standardsCount(input: StandardCountInput!): Int!
    }

    extend type Mutation {
        standardAdd(input: StandardInput!): Standard!
        standardUpdate(input: StandardInput!): Standard!
        standardDeleteMany(input: DeleteManyInput!): Count!
    }
`

export const resolvers = {
    StandardType: StandardType,
    StandardSortBy: StandardSortBy,
    Query: {
        standard: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard> | null> => {
            // Query database
            const dbModel = await StandardModel(prisma).findById(input, info);
            // Format data
            return dbModel ? StandardModel().toGraphQL(dbModel) : null;
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<any> => {
            // Create query for specified user
            const userQuery = input.userId ? { user: { id: input.userId } } : undefined;
            // return search query
            return await StandardModel(prisma).search({...userQuery,}, input, info);
        },
        standardsCount: async (_parent: undefined, { input }: IWrap<StandardCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await StandardModel(prisma).count({}, input);
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
            const dbModel = await StandardModel(prisma).create(input as any, info);
            // Format object to GraphQL type
            if (dbModel) return StandardModel().toGraphQL(dbModel);
            throw new CustomError(CODE.ErrorUnknown);
        },
        /**
         * Update standards you've created
         * @returns 
         */
        standardUpdate: async (_parent: undefined, { input }: IWrap<StandardInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Update object
            //const dbModel = await StandardModel(prisma).update(input, info);
            // Format to GraphQL type
            //return StandardModel().toGraphQL(dbModel);
            throw new CustomError(CODE.NotImplemented);
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
    }
}