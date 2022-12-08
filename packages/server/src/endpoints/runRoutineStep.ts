import { gql } from 'apollo-server-express';
import { RunStepStatus } from '@shared/consts';
import { RunRoutineStepSortBy } from './types';

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

    type RunRoutineStep {
        id: ID!
        order: Int!
        contextSwitches: Int!
        timeStarted: Date
        timeElapsed: Int
        timeCompleted: Date
        name: String!
        status: RunRoutineStepStatus!
        step: [Int!]!
        run: RunRoutine!
        node: Node
        subroutine: Routine
    }

    input RunRoutineStepCreateInput {
        id: ID!
        nodeId: ID
        contextSwitches: Int
        subroutineVersionId: ID
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

`

const objectType = 'RunRoutineStep';
export const resolvers: {
    RunRoutineStepSortBy: typeof RunRoutineStepSortBy;
    RunRoutineStepStatus: typeof RunStepStatus;
} = {
    RunRoutineStepSortBy,
    RunRoutineStepStatus: RunStepStatus,
}