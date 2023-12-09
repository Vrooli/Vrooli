import { gql } from "apollo-server-express";
import { EndpointsStatsSmartContract, StatsSmartContractEndpoints } from "../logic/statsSmartContract";

export const typeDef = gql`
    enum StatsSmartContractSortBy {
        PeriodStartAsc
        PeriodStartDesc
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
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        calls: Int!
        routineVersions: Int!
    }

    type Query {
        statsSmartContract(input: StatsSmartContractSearchInput!): StatsSmartContractSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsSmartContract["Query"]
} = {
    ...StatsSmartContractEndpoints,
};
