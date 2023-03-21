import { FindByIdInput, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceSearchInput, ScheduleRecurrenceSortBy, ScheduleRecurrenceType, ScheduleRecurrenceUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

export const typeDef = gql`
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
    }
    type ScheduleRecurrence {
        id: ID!
        recurrenceType: ScheduleRecurrenceType!
        interval: Int!
        dayOfWeek: Int
        dayOfMonth: Int
        month: Int
        endDate: Date
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
`

const objectType = 'ScheduleRecurrence';
export const resolvers: {
    ScheduleRecurrenceSortBy: typeof ScheduleRecurrenceSortBy;
    ScheduleRecurrenceType: typeof ScheduleRecurrenceType;
    Query: {
        scheduleRecurrence: GQLEndpoint<FindByIdInput, FindOneResult<ScheduleRecurrence>>;
        scheduleRecurrences: GQLEndpoint<ScheduleRecurrenceSearchInput, FindManyResult<ScheduleRecurrence>>;
    },
    Mutation: {
        scheduleRecurrenceCreate: GQLEndpoint<ScheduleRecurrenceCreateInput, CreateOneResult<ScheduleRecurrence>>;
        scheduleRecurrenceUpdate: GQLEndpoint<ScheduleRecurrenceUpdateInput, UpdateOneResult<ScheduleRecurrence>>;
    }
} = {
    ScheduleRecurrenceSortBy,
    ScheduleRecurrenceType,
    Query: {
        scheduleRecurrence: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        scheduleRecurrences: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        scheduleRecurrenceCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        scheduleRecurrenceUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}