import { RunRoutineStepSortBy, RunRoutineStepStatus } from "@local/shared";
import { gql } from "apollo-server-express";

export const typeDef = gql`
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
        step: [Int!]!
        timeElapsed: Int
        nodeConnect: ID
        subroutineVersionConnect: ID
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
        run: RunRoutine!
        node: Node
        subroutine: RoutineVersion
    }

`;

const objectType = "RunRoutineStep";
export const resolvers: {
    RunRoutineStepSortBy: typeof RunRoutineStepSortBy;
    RunRoutineStepStatus: typeof RunRoutineStepStatus;
} = {
    RunRoutineStepSortBy,
    RunRoutineStepStatus,
};
