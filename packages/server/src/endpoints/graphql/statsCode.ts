import { gql } from "apollo-server-express";
import { EndpointsStatsCode, StatsCodeEndpoints } from "../logic/statsCode";

export const typeDef = gql`
    enum StatsCodeSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsCodeSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsCodeSortBy
        take: Int
    }
    type StatsCodeSearchResult {
        pageInfo: PageInfo!
        edges: [StatsCodeEdge!]!
    }
    type StatsCodeEdge {
        cursor: String!
        node: StatsCode!
    }

    type StatsCode {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        calls: Int!
        routineVersions: Int!
    }

    type Query {
        statsCode(input: StatsCodeSearchInput!): StatsCodeSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsCode["Query"]
} = {
    ...StatsCodeEndpoints,
};
