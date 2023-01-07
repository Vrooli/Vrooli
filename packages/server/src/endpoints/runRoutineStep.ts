import { gql } from 'apollo-server-express';
import { RunRoutineStepStatus } from '@shared/consts';
import { RunRoutineStepSortBy } from '@shared/consts';

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
        nodeConnect: ID
        contextSwitches: Int
        subroutineVersionConnect: ID
        order: Int!
        step: [Int!]!
        timeElapsed: Int
        name: String!
    }
    input RunRoutineStepUpdateInput {
        id: ID!
        contextSwitches: Int
        status: RunRoutineStepStatus
        timeElapsed: Int
    }
    type RunRoutineStep {
        type: GqlModelType!
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
        subroutine: Routine
    }

`

const objectType = 'RunRoutineStep';
export const resolvers: {
    RunRoutineStepSortBy: typeof RunRoutineStepSortBy;
    RunRoutineStepStatus: typeof RunRoutineStepStatus;
} = {
    RunRoutineStepSortBy,
    RunRoutineStepStatus: RunRoutineStepStatus,
}