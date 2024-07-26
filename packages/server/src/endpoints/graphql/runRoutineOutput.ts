import { RunRoutineOutputSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRunRoutineOutput, RunRoutineOutputEndpoints } from "../logic/runRoutineOutput";

export const typeDef = gql`
    enum RunRoutineOutputSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input RunRoutineOutputCreateInput {
        id: ID!
        data: String!
        outputConnect: ID!
        runRoutineConnect: ID!
    }
    input RunRoutineOutputUpdateInput {
        id: ID!
        data: String!
    }
    type RunRoutineOutput {
        id: ID!
        data: String!
        output: RoutineVersionOutput!
        runRoutine: RunRoutine!
    }

    input RunRoutineOutputSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        runRoutineIds: [ID!]
        take: Int
        updatedTimeFrame: TimeFrame
    }
    type RunRoutineOutputSearchResult {
        pageInfo: PageInfo!
        edges: [RunRoutineOutputEdge!]!
    }
    type RunRoutineOutputEdge {
        cursor: String!
        node: RunRoutineOutput!
    }

    extend type Query {
        runRoutineOutputs(input: RunRoutineOutputSearchInput!): RunRoutineOutputSearchResult!
    }
`;

export const resolvers: {
    RunRoutineOutputSortBy: typeof RunRoutineOutputSortBy;
    Query: EndpointsRunRoutineOutput["Query"];
} = {
    RunRoutineOutputSortBy,
    ...RunRoutineOutputEndpoints,
};
