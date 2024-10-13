import { RunProjectStepStatus } from "@local/shared";

export const typeDef = `#graphql
    enum RunProjectStepStatus {
        InProgress
        Completed
        Skipped
    }

    input RunProjectStepCreateInput {
        id: ID!
        contextSwitches: Int
        name: String!
        order: Int!
        status: RunProjectStepStatus
        step: [Int!]!
        timeElapsed: Int
        directoryConnect: ID
        nodeConnect: ID
        runProjectConnect: ID!
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
        directory: ProjectVersionDirectory
        node: Node
        runProject: RunProject!
    }

`;

export const resolvers: {
    RunProjectStepStatus: typeof RunProjectStepStatus;
} = {
    RunProjectStepStatus,
};
