import { gql } from 'apollo-server-express';
import { IWrap } from '../types';
import { RunInputSortBy } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { readManyHelper, RunInputModel } from '../models';
import { rateLimit } from '../rateLimit';

export const typeDef = gql`
    enum RunInputSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    type RunInput {
        id: ID!
        data: String!
        input: [InputItem!]
    }

    input RunInputSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        routineIds: [ID!]
        standardIds: [ID!]
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type RunInputSearchResult {
        pageInfo: PageInfo!
        edges: [RunInputEdge!]!
    }

    type RunInputEdge {
        cursor: String!
        node: RunInput!
    }

    input RunInputCreateInput {
        id: ID!
        data: String!
    }

    input RunInputUpdateInput {
        id: ID!
        data: String!
    }

    extend type Query {
        runInputs(input: RunInputSearchInput!): RunInputSearchResult!
    }
`

export const resolvers = {
    RunInputSortBy: RunInputSortBy,
    Query: {
        runInputs: async (_parent: undefined, { input }: IWrap<any>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<any> => {
            await rateLimit({ info, max: 1000, req });
            return readManyHelper({ info, input, model: RunInputModel, prisma, userId: req.userId });
        },
    },
}