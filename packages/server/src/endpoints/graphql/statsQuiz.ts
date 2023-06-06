import { gql } from "apollo-server-express";
import { EndpointsStatsQuiz, StatsQuizEndpoints } from "../logic";

export const typeDef = gql`
    enum StatsQuizSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsQuizSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsQuizSortBy
        take: Int
    }
    type StatsQuizSearchResult {
        pageInfo: PageInfo!
        edges: [StatsQuizEdge!]!
    }
    type StatsQuizEdge {
        cursor: String!
        node: StatsQuiz!
    }

    type StatsQuiz {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        timesStarted: Int!
        timesPassed: Int!
        timesFailed: Int!
        scoreAverage: Float!
        completionTimeAverage: Float!
    }

    type Query {
        statsQuiz(input: StatsQuizSearchInput!): StatsQuizSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsQuiz["Query"];
} = {
    ...StatsQuizEndpoints,
};
