import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';
import { StatsSmartContractSearchInput, StatsSmartContractSearchResult } from '@shared/consts';

export const typeDef = gql`
    enum StatsSmartContractSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input StatsSmartContractSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsSmartContractSortBy
        take: Int
    }
    type StatsSmartContractSearchResult {
        pageInfo: PageInfo!
        edges: [StatsSmartContractEdge!]!
    }
    type StatsSmartContractEdge {
        cursor: String!
        node: StatsSmartContract!
    }

    type StatsSmartContract {
        id: ID!
        created_at: Date!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        calls: Int!
        routineVersions: Int!
    }

    type Query {
        statsSmartContract(input: StatsSmartContractSearchInput!): StatsSmartContractSearchResult!
    }
 `

const objectType = 'StatsSmartContract';
export const resolvers: {
    Query: {
        statsSmartContract: GQLEndpoint<StatsSmartContractSearchInput, StatsSmartContractSearchResult>;
    },
} = {
    Query: {
        statsSmartContract: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}