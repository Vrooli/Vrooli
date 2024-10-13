import { EndpointsStatsRoutine, StatsRoutineEndpoints } from "../logic/statsRoutine";

export const typeDef = `#graphql
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

export const resolvers: {
    Query: EndpointsStatsRoutine["Query"]
} = {
    ...StatsRoutineEndpoints,
};
