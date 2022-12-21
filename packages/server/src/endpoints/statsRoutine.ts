import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';

export const typeDef = gql`
    enum StatsRoutineSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input StatsRoutineSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsRoutineSortBy
        take: Int
    }
    type StatsRoutineSearchResult {
        pageInfo: PageInfo!
        edges: [StatsRoutineEdge!]!
    }
    type StatsRoutineEdge {
        cursor: String!
        node: StatsRoutine!
    }

    type StatsRoutine {
        id: ID!
        created_at: Date!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
    }

    type Query {
        statsRoutine(input: StatsRoutineSearchInput!): StatsRoutineSearchResult!
    }
 `

const objectType = 'StatsRoutine';
export const resolvers: {
    Query: {
        statsRoutine: GQLEndpoint<any, any>;
    },
} = {
    Query: {
        statsRoutine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}