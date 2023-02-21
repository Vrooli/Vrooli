import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Label, LabelSearchInput, RunRoutineScheduleSortBy, RunRoutineSchedule, RunRoutineScheduleSearchInput, RunRoutineScheduleCreateInput, RunRoutineScheduleUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RunRoutineScheduleSortBy {
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
        WindowStartAsc
        WindowStartDesc
        WindowEndAsc
        WindowEndDesc
    }

    input RunRoutineScheduleCreateInput {
        id: ID!
        attemptAutomatic: Boolean
        maxAutomaticAttempts: Int
        timeZone: String
        windowStart: Date
        windowEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        runRoutineConnect: ID!
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        translationsCreate: [RunRoutineScheduleTranslationCreateInput!]
    }
    input RunRoutineScheduleUpdateInput {
        id: ID!
        attemptAutomatic: Boolean
        maxAutomaticAttempts: Int
        timeZone: String
        windowStart: Date
        windowEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        runRoutineConnect: ID!
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        translationsCreate: [RunRoutineScheduleTranslationCreateInput!]
        translationsUpdate: [RunRoutineScheduleTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type RunRoutineSchedule {
        id: ID!
        created_at: Date!
        updated_at: Date!
        attemptAutomatic: Boolean!
        maxAutomaticAttempts: Int!
        timeZone: String
        windowStart: Date
        windowEnd: Date
        recurring: Boolean!
        recurrStart: Date
        recurrEnd: Date
        labels: [Label!]!
        runRoutine: RunRoutine!
        translations: [RunRoutineScheduleTranslation!]!
    }

    input RunRoutineScheduleTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input RunRoutineScheduleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type RunRoutineScheduleTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input RunRoutineScheduleSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        maxEventStart: Date
        maxEventEnd: Date
        maxRecurrStart: Date
        maxRecurrEnd: Date
        minEventStart: Date
        minEventEnd: Date
        minRecurrStart: Date
        minRecurrEnd: Date
        labelsIds: [ID!]
        runRoutineOrganizationId: ID # If not provided, uses your user ID
        searchString: String
        sortBy: RunRoutineScheduleSortBy
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RunRoutineScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [RunRoutineScheduleEdge!]!
    }

    type RunRoutineScheduleEdge {
        cursor: String!
        node: RunRoutineSchedule!
    }

    extend type Query {
        runRoutineSchedule(input: FindByIdInput!): RunRoutineSchedule
        runRoutineSchedules(input: RunRoutineScheduleSearchInput!): RunRoutineScheduleSearchResult!
    }

    extend type Mutation {
        runRoutineScheduleCreate(input: RunRoutineScheduleCreateInput!): RunRoutineSchedule!
        runRoutineScheduleUpdate(input: RunRoutineScheduleUpdateInput!): RunRoutineSchedule!
    }
`

const objectType = 'RunRoutineSchedule';
export const resolvers: {
    RunRoutineScheduleSortBy: typeof RunRoutineScheduleSortBy;
    Query: {
        runRoutineSchedule: GQLEndpoint<FindByIdInput, FindOneResult<RunRoutineSchedule>>;
        runRoutineSchedules: GQLEndpoint<RunRoutineScheduleSearchInput, FindManyResult<RunRoutineSchedule>>;
    },
    Mutation: {
        runRoutineScheduleCreate: GQLEndpoint<RunRoutineScheduleCreateInput, CreateOneResult<RunRoutineSchedule>>;
        runRoutineScheduleUpdate: GQLEndpoint<RunRoutineScheduleUpdateInput, UpdateOneResult<RunRoutineSchedule>>;
    }
} = {
    RunRoutineScheduleSortBy,
    Query: {
        runRoutineSchedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        runRoutineSchedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        runRoutineScheduleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runRoutineScheduleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}