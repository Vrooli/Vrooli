import { gql } from "apollo-server-express";
import { EndpointsStatsOrganization, StatsOrganizationEndpoints } from "../logic/statsOrganization";

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

export const resolvers: {
    Query: EndpointsStatsOrganization["Query"];
} = {
    ...StatsOrganizationEndpoints,
};
