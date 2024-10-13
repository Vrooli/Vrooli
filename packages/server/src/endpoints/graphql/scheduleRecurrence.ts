import { ScheduleRecurrenceSortBy, ScheduleRecurrenceType } from "@local/shared";
import { EndpointsScheduleRecurrence, ScheduleRecurrenceEndpoints } from "../logic/scheduleRecurrence";

export const typeDef = `#graphql
    enum ScheduleRecurrenceSortBy {
        DayOfWeekAsc
        DayOfWeekDesc
        DayOfMonthAsc
        DayOfMonthDesc
        MonthAsc
        MonthDesc
        EndDateAsc
        EndDateDesc
    }

    enum ScheduleRecurrenceType {
        Daily
        Weekly
        Monthly
        Yearly
    }

    input ScheduleRecurrenceCreateInput {
        id: ID!
        recurrenceType: ScheduleRecurrenceType!
        interval: Int!
        dayOfWeek: Int
        dayOfMonth: Int
        month: Int
        endDate: Date
        duration: Int!
        scheduleConnect: ID
        scheduleCreate: ScheduleCreateInput
    }
    input ScheduleRecurrenceUpdateInput {
        id: ID!
        recurrenceType: ScheduleRecurrenceType
        interval: Int
        dayOfWeek: Int
        dayOfMonth: Int
        month: Int
        endDate: Date
        duration: Int
    }
    type ScheduleRecurrence {
        id: ID!
        recurrenceType: ScheduleRecurrenceType!
        interval: Int!
        dayOfWeek: Int
        dayOfMonth: Int
        month: Int
        endDate: Date
        duration: Int!
        schedule: Schedule!
    }

    input ScheduleRecurrenceSearchInput {
        after: String
        dayOfWeek: Int
        dayOfMonth: Int
        endDateTimeFrame: TimeFrame
        ids: [ID!]
        interval: Int
        month: Int
        recurrenceType: ScheduleRecurrenceType
        searchString: String
        sortBy: ScheduleRecurrenceSortBy
        take: Int
    }

    type ScheduleRecurrenceSearchResult {
        pageInfo: PageInfo!
        edges: [ScheduleRecurrenceEdge!]!
    }

    type ScheduleRecurrenceEdge {
        cursor: String!
        node: ScheduleRecurrence!
    }

    extend type Query {
        scheduleRecurrence(input: FindByIdInput!): ScheduleRecurrence
        scheduleRecurrences(input: ScheduleRecurrenceSearchInput!): ScheduleRecurrenceSearchResult!
    }

    extend type Mutation {
        scheduleRecurrenceCreate(input: ScheduleRecurrenceCreateInput!): ScheduleRecurrence!
        scheduleRecurrenceUpdate(input: ScheduleRecurrenceUpdateInput!): ScheduleRecurrence!
    }
`;

export const resolvers: {
    ScheduleRecurrenceSortBy: typeof ScheduleRecurrenceSortBy;
    ScheduleRecurrenceType: typeof ScheduleRecurrenceType;
    Query: EndpointsScheduleRecurrence["Query"];
    Mutation: EndpointsScheduleRecurrence["Mutation"];
} = {
    ScheduleRecurrenceSortBy,
    ScheduleRecurrenceType,
    ...ScheduleRecurrenceEndpoints,
};
