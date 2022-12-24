import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';
import { StatsUserSearchInput, StatsUserSearchResult } from '@shared/consts';

export const typeDef = gql`
    enum StatsUserSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input StatsUserSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsUserSortBy
        take: Int
    }
    type StatsUserSearchResult {
        pageInfo: PageInfo!
        edges: [StatsUserEdge!]!
    }
    type StatsUserEdge {
        cursor: String!
        node: StatsUser!
    }

    type StatsUser {
        id: ID!
        created_at: Date!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
    }

    type Query {
        statsUser(input: StatsUserSearchInput!): StatsUserSearchResult!
    }
 `

const objectType = 'StatsUser';
export const resolvers: {
    Query: {
        statsUser: GQLEndpoint<StatsUserSearchInput, StatsUserSearchResult>;
    },
} = {
    Query: {
        statsUser: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}