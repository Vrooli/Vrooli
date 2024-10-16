import { RunRoutineStepSortBy, RunRoutineStepStatus } from "@local/shared";

export const typeDef = `#graphql
    enum RunRoutineStepSortBy {
        ContextSwitchesAsc
        ContextSwitchesDesc
        OrderAsc
        OrderDesc
        TimeCompletedAsc
        TimeCompletedDesc
        TimeStartedAsc
        TimeStartedDesc
        TimeElapsedAsc
        TimeElapsedDesc
    }

    enum RunRoutineStepStatus {
        InProgress
        Completed
        Skipped
    }

    input RunRoutineStepCreateInput {
        id: ID!
        contextSwitches: Int
        name: String!
        order: Int!
        status: RunRoutineStepStatus
        step: [Int!]!
        timeElapsed: Int
        nodeConnect: ID
        runRoutineConnect: ID!
        subroutineConnect: ID
    }
    input RunRoutineStepUpdateInput {
        id: ID!
        contextSwitches: Int
        status: RunRoutineStepStatus
        timeElapsed: Int
    }
    type RunRoutineStep {
        id: ID!
        order: Int!
        contextSwitches: Int!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        name: String!
        status: RunRoutineStepStatus!
        step: [Int!]!
        node: Node
        runRoutine: RunRoutine!
        subroutine: RoutineVersion
    }

`;

export const resolvers: {
    RunRoutineStepSortBy: typeof RunRoutineStepSortBy;
    RunRoutineStepStatus: typeof RunRoutineStepStatus;
} = {
    RunRoutineStepSortBy,
    RunRoutineStepStatus,
};
