import { RunProjectStepStatus } from "@local/shared";
import { gql } from "apollo-server-express";

export const typeDef = gql`
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

`;

export const resolvers: {
    RunProjectStepStatus: typeof RunProjectStepStatus;
} = {
    RunProjectStepStatus,
};
