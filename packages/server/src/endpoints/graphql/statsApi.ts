import { gql } from "apollo-server-express";
import { EndpointsStatsApi, StatsApiEndpoints } from "../logic/statsApi";

export const typeDef = gql`
    enum StatsApiSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsApiSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsApiSortBy
        take: Int
    }
    type StatsApiSearchResult {
        pageInfo: PageInfo!
        edges: [StatsApiEdge!]!
    }
    type StatsApiEdge {
        cursor: String!
        node: StatsApi!
    }

    type StatsApi {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        calls: Int!
        routineVersions: Int!
    }

    type Query {
        statsApi(input: StatsApiSearchInput!): StatsApiSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsApi["Query"];
} = {
    ...StatsApiEndpoints,
};
