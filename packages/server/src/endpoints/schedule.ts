import { FindVersionInput, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleSortBy, ScheduleUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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

    input ScheduleCreateInput {
        id: ID!
        startTime: Date
        endTime: Date
        timezone: String!
        exceptionsCreate: [ScheduleExceptionCreateInput!]
        focusModesConnect: [ID!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        meetingsConnect: [ID!]
        recurrencesCreate: [ScheduleRecurrenceCreateInput!]
        runProjectsConnect: [ID!]
        runRoutinesConnect: [ID!]
        translationsCreate: [ScheduleTranslationCreateInput!]
    }
    input ScheduleUpdateInput {
        id: ID!
        startTime: Date
        endTime: Date
        timezone: String
        exceptionsCreate: [ScheduleExceptionCreateInput!]
        exceptionsUpdate: [ScheduleExceptionUpdateInput!]
        exceptionsDelete: [ID!]
        focusModesConnect: [ID!]
        focusModesDisconnect: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        meetingsConnect: [ID!]
        meetingsDisconnect: [ID!]
        recurrencesCreate: [ScheduleRecurrenceCreateInput!]
        recurrencesUpdate: [ScheduleRecurrenceUpdateInput!]
        recurrencesDelete: [ID!]
        runProjectsConnect: [ID!]
        runProjectsDisconnect: [ID!]
        runRoutinesConnect: [ID!]
        runRoutinesDisconnect: [ID!]
        translationsCreate: [ScheduleTranslationCreateInput!]
        translationsUpdate: [ScheduleTranslationUpdateInput!]
        translationsDelete: [ID!]
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
        translations: [ScheduleTranslation!]!
    }

    input ScheduleTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input ScheduleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ScheduleTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input ScheduleSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        endTimeFrame: TimeFrame
        ids: [ID!]
        translationLanguages: [String!]
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
`

const objectType = 'Schedule';
export const resolvers: {
    ScheduleSortBy: typeof ScheduleSortBy;
    Query: {
        schedule: GQLEndpoint<FindVersionInput, FindOneResult<Schedule>>;
        schedules: GQLEndpoint<ScheduleSearchInput, FindManyResult<Schedule>>;
    },
    Mutation: {
        scheduleCreate: GQLEndpoint<ScheduleCreateInput, CreateOneResult<Schedule>>;
        scheduleUpdate: GQLEndpoint<ScheduleUpdateInput, UpdateOneResult<Schedule>>;
    }
} = {
    ScheduleSortBy,
    Query: {
        schedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        schedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        scheduleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        scheduleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}