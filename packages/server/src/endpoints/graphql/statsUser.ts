import { EndpointsStatsUser, StatsUserEndpoints } from "../logic/statsUser";

export const typeDef = `#graphql
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
        codesCreated: Int!
        codesCompleted: Int!
        codeCompletionTimeAverage: Float!
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
        standardsCreated: Int!
        standardsCompleted: Int!
        standardCompletionTimeAverage: Float!
        teamssCreated: Int!
    }

    type Query {
        statsUser(input: StatsUserSearchInput!): StatsUserSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsUser["Query"]
} = {
    ...StatsUserEndpoints,
};
