import { gql } from 'apollo-server-express';
import { FindManyResult, GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';
import { RunRoutineInput, RunRoutineInputSearchInput, RunRoutineInputSortBy } from './types';

export const typeDef = gql`
    enum RunRoutineInputSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    type RunRoutineInput {
        id: ID!
        data: String!
        input: RoutineVersionInput!
    }

    input RunRoutineInputSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        routineIds: [ID!]
        standardIds: [ID!]
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type RunRoutineInputSearchResult {
        pageInfo: PageInfo!
        edges: [RunRoutineInputEdge!]!
    }

    type RunRoutineInputEdge {
        cursor: String!
        node: RunRoutineInput!
    }

    input RunRoutineInputCreateInput {
        id: ID!
        data: String!
        inputId: ID!
    }

    input RunRoutineInputUpdateInput {
        id: ID!
        data: String!
    }

    extend type Query {
        runRoutineInputs(input: RunRoutineInputSearchInput!): RunRoutineInputSearchResult!
    }
`

const objectType = 'RunRoutineInput';
export const resolvers: {
    RunRoutineInputSortBy: typeof RunRoutineInputSortBy;
    Query: {
        runRoutineInputs: GQLEndpoint<RunRoutineInputSearchInput, FindManyResult<RunRoutineInput>>;
    },
} = {
    RunRoutineInputSortBy,
    Query: {
        runRoutineInputs: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}