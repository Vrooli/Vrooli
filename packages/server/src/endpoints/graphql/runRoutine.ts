import { RunRoutineSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRunRoutine, RunRoutineEndpoints } from "../logic/runRoutine";

export const typeDef = gql`
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
        stepsCreate: [RunRoutineStepCreateInput!]
        inputsCreate: [RunRoutineInputCreateInput!]
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
        routineVersion: RoutineVersion
        runProject: RunProject
        schedule: Schedule
        steps: [RunRoutineStep!]!
        stepsCount: Int!
        inputs: [RunRoutineInput!]!
        inputsCount: Int!
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

    input RunRoutineCompleteInput {
        id: ID! # Run ID if "exists" is true, or routine version ID if "exists" is false
        completedComplexity: Int # Even though the runRoutine was completed, the user may not have completed every subroutine
        exists: Boolean # If true, runRoutine ID is provided, otherwise routine ID so we can create a runRoutine
        name: String # Title of routine, so runRoutine name stays consistent even if routine updates/deletes
        finalStepCreate: RunRoutineStepCreateInput
        finalStepUpdate: RunRoutineStepUpdateInput
        inputsDelete: [ID!]
        inputsCreate: [RunRoutineInputCreateInput!]
        inputsUpdate: [RunRoutineInputUpdateInput!]
        routineVersionConnect: ID # Only needed if "exists" is false
        wasSuccessful: Boolean
    }
    input RunRoutineCancelInput {
        id: ID!
    }

    extend type Query {
        runRoutine(input: FindByIdInput!): RunRoutine
        runRoutines(input: RunRoutineSearchInput!): RunRoutineSearchResult!
    }

    extend type Mutation {
        runRoutineCreate(input: RunRoutineCreateInput!): RunRoutine!
        runRoutineUpdate(input: RunRoutineUpdateInput!): RunRoutine!
        runRoutineDeleteAll: Count!
        runRoutineComplete(input: RunRoutineCompleteInput!): RunRoutine!
        runRoutineCancel(input: RunRoutineCancelInput!): RunRoutine!
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
