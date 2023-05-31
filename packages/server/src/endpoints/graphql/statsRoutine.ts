import { StatsRoutineSearchInput, StatsRoutineSearchResult } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { GQLEndpoint } from "../../types";

export const typeDef = gql`
    enum StatsRoutineSortBy {
        PeriodStartAsc
        PeriodStartDesc
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
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        runsStarted: Int!
        runsCompleted: Int!
        runCompletionTimeAverage: Float!
        runContextSwitchesAverage: Float!
    }

    type Query {
        statsRoutine(input: StatsRoutineSearchInput!): StatsRoutineSearchResult!
    }
 `;

const objectType = "StatsRoutine";
export const resolvers: {
    Query: {
        statsRoutine: GQLEndpoint<StatsRoutineSearchInput, StatsRoutineSearchResult>;
    },
} = {
    Query: {
        statsRoutine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
