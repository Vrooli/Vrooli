import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { rateLimit } from "../middleware";
export const typeDef = gql `
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
const objectType = "StatsQuiz";
export const resolvers = {
    Query: {
        statsQuiz: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
//# sourceMappingURL=statsQuiz.js.map