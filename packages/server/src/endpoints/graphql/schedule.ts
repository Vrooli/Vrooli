import { ScheduleFor, ScheduleSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsSchedule, ScheduleEndpoints } from "../logic";

export const typeDef = gql`
    enum ScheduleSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StartTimeAsc
        StartTimeDesc
        EndTimeAsc
        EndTimeDesc
    }

    enum ScheduleFor {
        FocusMode
        Meeting
        RunProject
        RunRoutine
    }

    input ScheduleCreateInput {
        id: ID!
        startTime: Date
        endTime: Date
        timezone: String!
        exceptionsCreate: [ScheduleExceptionCreateInput!]
        focusModeConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        meetingConnect: ID
        recurrencesCreate: [ScheduleRecurrenceCreateInput!]
        runProjectConnect: ID
        runRoutineConnect: ID
    }
    input ScheduleUpdateInput {
        id: ID!
        startTime: Date
        endTime: Date
        timezone: String
        exceptionsCreate: [ScheduleExceptionCreateInput!]
        exceptionsUpdate: [ScheduleExceptionUpdateInput!]
        exceptionsDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        recurrencesCreate: [ScheduleRecurrenceCreateInput!]
        recurrencesUpdate: [ScheduleRecurrenceUpdateInput!]
        recurrencesDelete: [ID!]
    }
    type Schedule {
        id: ID!
        created_at: Date!
        updated_at: Date!
        startTime: Date!
        endTime: Date!
        timezone: String!
        exceptions: [ScheduleException!]!
        focusModes: [FocusMode!]!
        labels: [Label!]!
        meetings: [Meeting!]!
        recurrences: [ScheduleRecurrence!]!
        runProjects: [RunProject!]!
        runRoutines: [RunRoutine!]!
    }

    input ScheduleSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        endTimeFrame: TimeFrame
        ids: [ID!]
        scheduleFor: ScheduleFor
        scheduleForUserId: ID
        searchString: String
        sortBy: ScheduleSortBy
        startTimeFrame: TimeFrame
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [ScheduleEdge!]!
    }

    type ScheduleEdge {
        cursor: String!
        node: Schedule!
    }

    extend type Query {
        schedule(input: FindVersionInput!): Schedule
        schedules(input: ScheduleSearchInput!): ScheduleSearchResult!
    }

    extend type Mutation {
        scheduleCreate(input: ScheduleCreateInput!): Schedule!
        scheduleUpdate(input: ScheduleUpdateInput!): Schedule!
    }
`;

export const resolvers: {
    ScheduleSortBy: typeof ScheduleSortBy;
    ScheduleFor: typeof ScheduleFor;
    Query: EndpointsSchedule["Query"];
    Mutation: EndpointsSchedule["Mutation"];
} = {
    ScheduleSortBy,
    ScheduleFor,
    ...ScheduleEndpoints,
};
