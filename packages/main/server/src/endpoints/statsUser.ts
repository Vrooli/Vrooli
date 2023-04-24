import { StatsUserSearchInput, StatsUserSearchResult } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { rateLimit } from "../middleware";
import { GQLEndpoint } from "../types";

export const typeDef = gql`
    enum StatsUserSortBy {
        PeriodStartAsc
        PeriodStartDesc
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
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        apisCreated: Int!
        organizationsCreated: Int!
        projectsCreated: Int!
        projectsCompleted: Int!
        projectCompletionTimeAverage: Float!
        quizzesPassed: Int!
        quizzesFailed: Int!
        routinesCreated: Int!
        routinesCompleted: Int!
        routineCompletionTimeAverage: Float!
        runProjectsStarted: Int!
        runProjectsCompleted: Int!
        runProjectCompletionTimeAverage: Float!
        runProjectContextSwitchesAverage: Float!
        runRoutinesStarted: Int!
        runRoutinesCompleted: Int!
        runRoutineCompletionTimeAverage: Float!
        runRoutineContextSwitchesAverage: Float!
        smartContractsCreated: Int!
        smartContractsCompleted: Int!
        smartContractCompletionTimeAverage: Float!
        standardsCreated: Int!
        standardsCompleted: Int!
        standardCompletionTimeAverage: Float!
    }

    type Query {
        statsUser(input: StatsUserSearchInput!): StatsUserSearchResult!
    }
 `;

const objectType = "StatsUser";
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
};
