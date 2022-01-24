import { gql } from 'apollo-server-express';
import { CODE, StandardSortBy } from '@local/shared';
import { CustomError } from '../error';
import { StandardModel } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { DeleteManyInput, DeleteOneInput, FindByIdInput, Standard, StandardCountInput, StandardInput, StandardSearchInput, Success } from './types';
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
        organizationId: ID
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
        isStarred: Boolean
        score: Int!
        isUpvoted: Boolean
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
        standardDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    StandardType: StandardType,
    StandardSortBy: StandardSortBy,
    Query: {
        standard: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard> | null> => {
            const data = await StandardModel(prisma).findStandard(req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<any> => {
            const data = await StandardModel(prisma).searchStandards({}, req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
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
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await StandardModel(prisma).addStandard(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        /**
         * Delete a standard you've created. Other standards must go through a reporting system
         * @returns 
         */
        standardDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await StandardModel(prisma).deleteStandard(req.userId, input);
        },
    }
}