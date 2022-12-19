import { gql } from 'apollo-server-express';
import { RunStepStatus } from '@shared/consts';
import { RunProjectStepSortBy } from './types';

export const typeDef = gql`
    enum RunProjectStepSortBy {
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

    enum RunProjectStepStatus {
        InProgress
        Completed
        Skipped
    }

    input RunProjectStepCreateInput {
        id: ID!
        nodeConnect: ID
        contextSwitches: Int
        directoryConnect: ID
        order: Int!
        step: [Int!]!
        timeElapsed: Int
        name: String!
    }
    input RunProjectStepUpdateInput {
        id: ID!
        contextSwitches: Int
        status: RunProjectStepStatus
        timeElapsed: Int
    }
    type RunProjectStep {
        id: ID!
        order: Int!
        contextSwitches: Int!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        name: String!
        status: RunProjectStepStatus!
        step: [Int!]!
        run: RunProject!
        node: Node
        directory: ProjectVersionDirectory
    }

`

const objectType = 'RunProjectStep';
export const resolvers: {
    RunProjectStepSortBy: typeof RunProjectStepSortBy;
    RunProjectStepStatus: typeof RunStepStatus;
} = {
    RunProjectStepSortBy,
    RunProjectStepStatus: RunStepStatus,
}