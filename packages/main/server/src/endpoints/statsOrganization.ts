import { StatsOrganizationSearchInput, StatsOrganizationSearchResult } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { rateLimit } from "../middleware";
import { GQLEndpoint } from "../types";

export const typeDef = gql`
    enum StatsOrganizationSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsOrganizationSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsOrganizationSortBy
        take: Int
    }
    type StatsOrganizationSearchResult {
        pageInfo: PageInfo!
        edges: [StatsOrganizationEdge!]!
    }
    type StatsOrganizationEdge {
        cursor: String!
        node: StatsOrganization!
    }

    type StatsOrganization {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        apis: Int!
        members: Int!
        notes: Int!
        projects: Int!
        routines: Int!
        runRoutinesStarted: Int!
        runRoutinesCompleted: Int!
        runRoutineCompletionTimeAverage: Float!
        runRoutineContextSwitchesAverage: Float!
        smartContracts: Int!
        standards: Int!
    }

    type Query {
        statsOrganization(input: StatsOrganizationSearchInput!): StatsOrganizationSearchResult!
    }
 `;

const objectType = "StatsOrganization";
export const resolvers: {
    Query: {
        statsOrganization: GQLEndpoint<StatsOrganizationSearchInput, StatsOrganizationSearchResult>;
    },
} = {
    Query: {
        statsOrganization: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
