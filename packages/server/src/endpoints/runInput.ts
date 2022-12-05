import { gql } from 'apollo-server-express';
import { FindManyResult, GQLEndpoint } from '../types';
import { RunInput, RunInputSearchInput, RunInputSortBy } from './types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';

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
        input: InputItem!
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
        inputId: ID!
    }

    input RunInputUpdateInput {
        id: ID!
        data: String!
    }

    extend type Query {
        runInputs(input: RunInputSearchInput!): RunInputSearchResult!
    }
`

const objectType = 'RunInput';
export const resolvers: {
    RunInputSortBy: typeof RunInputSortBy;
    Query: {
        runInputs: GQLEndpoint<RunInputSearchInput, FindManyResult<RunInput>>;
    },
} = {
    RunInputSortBy: RunInputSortBy,
    Query: {
        runInputs: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}