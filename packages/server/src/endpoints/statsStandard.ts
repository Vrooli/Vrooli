import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';
import { StatsStandardSearchInput, StatsStandardSearchResult } from '@shared/consts';

export const typeDef = gql`
    enum StatsStandardSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input StatsStandardSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsStandardSortBy
        take: Int
    }
    type StatsStandardSearchResult {
        pageInfo: PageInfo!
        edges: [StatsStandardEdge!]!
    }
    type StatsStandardEdge {
        cursor: String!
        node: StatsStandard!
    }

    type StatsStandard {
        id: ID!
        created_at: Date!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
    }

    type Query {
        statsStandard(input: StatsStandardSearchInput!): StatsStandardSearchResult!
    }
 `

const objectType = 'StatsStandard';
export const resolvers: {
    Query: {
        statsStandard: GQLEndpoint<StatsStandardSearchInput, StatsStandardSearchResult>;
    },
} = {
    Query: {
        statsStandard: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}