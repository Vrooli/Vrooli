import { RunRoutineInput, RunRoutineInputSearchInput, RunRoutineInputSortBy } from "@local/shared;";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { rateLimit } from "../middleware";
import { FindManyResult, GQLEndpoint } from "../types";

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
        routineIds: [ID!]
        standardIds: [ID!]
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

const objectType = "RunRoutineInput";
export const resolvers: {
    RunRoutineInputSortBy: typeof RunRoutineInputSortBy;
    Query: {
        runRoutineInputs: GQLEndpoint<RunRoutineInputSearchInput, FindManyResult<RunRoutineInput>>;
    },
} = {
    RunRoutineInputSortBy,
    Query: {
        runRoutineInputs: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
