import { RunRoutineInputSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRunRoutineInput, RunRoutineInputEndpoints } from "../logic/runRoutineInput";

export const typeDef = gql`
    enum RunRoutineInputSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input RunRoutineInputCreateInput {
        id: ID!
        data: String!
        inputConnect: ID!
        runRoutineConnect: ID!
    }
    input RunRoutineInputUpdateInput {
        id: ID!
        data: String!
    }
    type RunRoutineInput {
        id: ID!
        data: String!
        input: RoutineVersionInput!
        runRoutine: RunRoutine!
    }

    input RunRoutineInputSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        runRoutineIds: [ID!]
        take: Int
        updatedTimeFrame: TimeFrame
    }
    type RunRoutineInputSearchResult {
        pageInfo: PageInfo!
        edges: [RunRoutineInputEdge!]!
    }
    type RunRoutineInputEdge {
        cursor: String!
        node: RunRoutineInput!
    }

    extend type Query {
        runRoutineInputs(input: RunRoutineInputSearchInput!): RunRoutineInputSearchResult!
    }
`;

export const resolvers: {
    RunRoutineInputSortBy: typeof RunRoutineInputSortBy;
    Query: EndpointsRunRoutineInput["Query"];
} = {
    RunRoutineInputSortBy,
    ...RunRoutineInputEndpoints,
};
