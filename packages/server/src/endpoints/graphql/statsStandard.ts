import { gql } from "apollo-server-express";
import { EndpointsStatsStandard, StatsStandardEndpoints } from "../logic";

export const typeDef = gql`
    enum StatsStandardSortBy {
        PeriodStartAsc
        PeriodStartDesc
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
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        linksToInputs: Int!
        linksToOutputs: Int!
    }

    type Query {
        statsStandard(input: StatsStandardSearchInput!): StatsStandardSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsStandard["Query"]
} = {
    ...StatsStandardEndpoints,
};
