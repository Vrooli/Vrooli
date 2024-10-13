import { EndpointsStatsTeam, StatsTeamEndpoints } from "../logic/statsTeam";

export const typeDef = `#graphql
    enum StatsTeamSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsTeamSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsTeamSortBy
        take: Int
    }
    type StatsTeamSearchResult {
        pageInfo: PageInfo!
        edges: [StatsTeamEdge!]!
    }
    type StatsTeamEdge {
        cursor: String!
        node: StatsTeam!
    }

    type StatsTeam {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        apis: Int!
        codes: Int!
        members: Int!
        notes: Int!
        projects: Int!
        routines: Int!
        runRoutinesStarted: Int!
        runRoutinesCompleted: Int!
        runRoutineCompletionTimeAverage: Float!
        runRoutineContextSwitchesAverage: Float!
        standards: Int!
    }

    type Query {
        statsTeam(input: StatsTeamSearchInput!): StatsTeamSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsTeam["Query"];
} = {
    ...StatsTeamEndpoints,
};
