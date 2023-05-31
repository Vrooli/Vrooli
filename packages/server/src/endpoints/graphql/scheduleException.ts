import { ScheduleExceptionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsScheduleException, ScheduleExceptionEndpoints } from "../logic";

export const typeDef = gql`
    enum ScheduleExceptionSortBy {
        OriginalStartTimeAsc
        OriginalStartTimeDesc
        NewStartTimeAsc
        NewStartTimeDesc
        NewEndTimeAsc
        NewEndTimeDesc
    }

    input ScheduleExceptionCreateInput {
        id: ID!
        originalStartTime: Date!
        newStartTime: Date
        newEndTime: Date
        scheduleConnect: ID!
    }
    input ScheduleExceptionUpdateInput {
        id: ID!
        originalStartTime: Date
        newStartTime: Date
        newEndTime: Date
    }
    type ScheduleException {
        id: ID!
        originalStartTime: Date!
        newStartTime: Date
        newEndTime: Date
        schedule: Schedule!
    }

    input ScheduleExceptionSearchInput {
        after: String
        ids: [ID!]
        newStartTimeFrame: TimeFrame
        newEndTimeFrame: TimeFrame
        originalStartTimeFrame: TimeFrame
        searchString: String
        sortBy: ScheduleExceptionSortBy
        take: Int
    }

    type ScheduleExceptionSearchResult {
        pageInfo: PageInfo!
        edges: [ScheduleExceptionEdge!]!
    }

    type ScheduleExceptionEdge {
        cursor: String!
        node: ScheduleException!
    }

    extend type Query {
        scheduleException(input: FindByIdInput!): ScheduleException
        scheduleExceptions(input: ScheduleExceptionSearchInput!): ScheduleExceptionSearchResult!
    }

    extend type Mutation {
        scheduleExceptionCreate(input: ScheduleExceptionCreateInput!): ScheduleException!
        scheduleExceptionUpdate(input: ScheduleExceptionUpdateInput!): ScheduleException!
    }
`;

export const resolvers: {
    ScheduleExceptionSortBy: typeof ScheduleExceptionSortBy;
    Query: EndpointsScheduleException["Query"];
    Mutation: EndpointsScheduleException["Mutation"];
} = {
    ScheduleExceptionSortBy,
    ...ScheduleExceptionEndpoints,
};
