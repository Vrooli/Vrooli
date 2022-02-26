import { gql } from 'apollo-server-express';
import { CODE, StandardSortBy } from '@local/shared';
import { CustomError } from '../error';
import { createHelper, deleteOneHelper, readManyHelper, readOneHelper, StandardModel, standardSearcher, updateHelper } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Standard, StandardCountInput, StandardCreateInput, StandardUpdateInput, StandardSearchInput, Success, StandardSearchResult } from './types';
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

    input StandardCreateInput {
        default: String
        description: String
        isFile: Boolean
        name: String!
        schema: String
        type: StandardType
        version: String
        createdByUserId: ID
        createdByOrganizationId: ID
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    input StandardUpdateInput {
        id: ID!
        description: String
        makeAnonymous: Boolean
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    type Standard {
        id: ID!
        created_at: Date!
        updated_at: Date!
        default: String
        description: String
        name: String!
        isFile: Boolean!
        isStarred: Boolean!
        role: MemberRole
        isUpvoted: Boolean
        schema: String!
        score: Int!
        stars: Int!
        type: StandardType!
        comments: [Comment!]!
        creator: Contributor
        reports: [Report!]!
        routineInputs: [Routine!]!
        routineOutputs: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
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
        standardCreate(input: StandardCreateInput!): Standard!
        standardUpdate(input: StandardUpdateInput!): Standard!
        standardDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    StandardType: StandardType,
    StandardSortBy: StandardSortBy,
    Query: {
        standard: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard> | null> => {
            return readOneHelper(req.userId, input, info, prisma);
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<StandardSearchResult> => {
            return readManyHelper(req.userId, input, info, prisma, standardSearcher());
        },
        standardsCount: async (_parent: undefined, { input }: IWrap<StandardCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await StandardModel(prisma).count({}, input);
        },
    },
    Mutation: {
        /**
         * Create a new standard
         * @returns Standard object if successful
         */
        standardCreate: async (_parent: undefined, { input }: IWrap<StandardCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            return createHelper(req.userId, input, info, StandardModel(prisma).cud);
        },
        /**
         * Update a standard you created.
         * NOTE: You can only update the description and tags. If you need to update 
         * the other fields, you must either create a new standard (could be the same but with an updated
         * version number) or delete the old one and create a new one.
         * @returns Standard object if successful
         */
        standardUpdate: async (_parent: undefined, { input }: IWrap<StandardUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            return updateHelper(req.userId, input, info, StandardModel(prisma).cud);
        },
        /**
         * Delete a standard you've created. Other standards must go through a reporting system
         * @returns 
         */
        standardDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, StandardModel(prisma).cud);
        },
    }
}