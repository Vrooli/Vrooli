import { RunRoutineSortBy } from "@local/shared";
import { EndpointsRunRoutine, RunRoutineEndpoints } from "../logic/runRoutine";

export const typeDef = `#graphql
    enum RunRoutineSortBy {
        ContextSwitchesAsc
        ContextSwitchesDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateStartedAsc
        DateStartedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StepsAsc
        StepsDesc
    }

    input RunRoutineCreateInput {
        id: ID!
        isPrivate: Boolean!
        completedComplexity: Int
        contextSwitches: Int
        name: String!
        status: RunStatus!
        timeElapsed: Int
        stepsCreate: [RunRoutineStepCreateInput!]
        inputsCreate: [RunRoutineInputCreateInput!]
        outputsCreate: [RunRoutineOutputCreateInput!]
        scheduleCreate: ScheduleCreateInput
        routineVersionConnect: ID!
        runProjectConnect: ID
        teamConnect: ID
    }
    input RunRoutineUpdateInput {
        id: ID!
        isPrivate: Boolean
        completedComplexity: Int # Total completed complexity, including what was completed before this update
        contextSwitches: Int # Total contextSwitches, including what was completed before this update
        status: RunStatus
        timeElapsed: Int # Total time elapsed, including what was completed before this update
        stepsDelete: [ID!]
        stepsCreate: [RunRoutineStepCreateInput!]
        stepsUpdate: [RunRoutineStepUpdateInput!]
        inputsDelete: [ID!]
        inputsCreate: [RunRoutineInputCreateInput!]
        inputsUpdate: [RunRoutineInputUpdateInput!]
        outputsDelete: [ID!]
        outputsCreate: [RunRoutineOutputCreateInput!]
        outputsUpdate: [RunRoutineOutputUpdateInput!]
        scheduleCreate: ScheduleCreateInput
        scheduleUpdate: ScheduleUpdateInput
    }
    type RunRoutine {
        id: ID!
        isPrivate: Boolean!
        completedComplexity: Int!
        contextSwitches: Int!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        lastStep: [Int!]
        name: String!
        status: RunStatus!
        wasRunAutomatically: Boolean!
        inputs: [RunRoutineInput!]!
        outputs: [RunRoutineOutput!]!
        routineVersion: RoutineVersion
        runProject: RunProject
        schedule: Schedule
        steps: [RunRoutineStep!]!
        inputsCount: Int!
        outputsCount: Int!
        stepsCount: Int!
        team: Team
        user: User
        you: RunRoutineYou!
    }

    type RunRoutineYou {
        canDelete: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
    }

    input RunRoutineSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        scheduleStartTimeFrame: TimeFrame
        scheduleEndTimeFrame: TimeFrame
        status: RunStatus
        statuses: [RunStatus!]
        routineVersionId: ID
        searchString: String
        sortBy: RunRoutineSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RunRoutineSearchResult {
        pageInfo: PageInfo!
        edges: [RunRoutineEdge!]!
    }

    type RunRoutineEdge {
        cursor: String!
        node: RunRoutine!
    }

    extend type Query {
        runRoutine(input: FindByIdInput!): RunRoutine
        runRoutines(input: RunRoutineSearchInput!): RunRoutineSearchResult!
    }

    extend type Mutation {
        runRoutineCreate(input: RunRoutineCreateInput!): RunRoutine!
        runRoutineUpdate(input: RunRoutineUpdateInput!): RunRoutine!
        runRoutineDeleteAll: Count!
    }
`;

export const resolvers: {
    RunRoutineSortBy: typeof RunRoutineSortBy;
    Query: EndpointsRunRoutine["Query"];
    Mutation: EndpointsRunRoutine["Mutation"];
} = {
    RunRoutineSortBy,
    ...RunRoutineEndpoints,
};
