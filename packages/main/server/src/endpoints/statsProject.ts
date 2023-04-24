import { StatsProjectSearchInput, StatsProjectSearchResult } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { rateLimit } from "../middleware";
import { GQLEndpoint } from "../types";

export const typeDef = gql`
    enum StatsProjectSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsProjectSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsProjectSortBy
        take: Int
    }
    type StatsProjectSearchResult {
        pageInfo: PageInfo!
        edges: [StatsProjectEdge!]!
    }
    type StatsProjectEdge {
        cursor: String!
        node: StatsProject!
    }

    type StatsProject {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        directories: Int!
        apis: Int!
        notes: Int!
        organizations: Int!
        projects: Int!
        routines: Int!
        smartContracts: Int!
        standards: Int!
        runsStarted: Int!
        runsCompleted: Int!
        runCompletionTimeAverage: Float!
        runContextSwitchesAverage: Float!
    }

    type Query {
        statsProject(input: StatsProjectSearchInput!): StatsProjectSearchResult!
    }
 `;

const objectType = "StatsProject";
export const resolvers: {
    Query: {
        statsProject: GQLEndpoint<StatsProjectSearchInput, StatsProjectSearchResult>;
    },
} = {
    Query: {
        statsProject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
