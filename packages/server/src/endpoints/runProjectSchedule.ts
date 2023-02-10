import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Label, LabelSearchInput, RunProjectScheduleSortBy, RunProjectSchedule, RunProjectScheduleSearchInput, RunProjectScheduleCreateInput, RunProjectScheduleUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RunProjectScheduleSortBy {
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
        WindowStartAsc
        WindowStartDesc
        WindowEndAsc
        WindowEndDesc
    }

    input RunProjectScheduleCreateInput {
        id: ID!
        timeZone: String
        windowStart: Date
        windowEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        runProjectConnect: ID!
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        translationsCreate: [RunProjectScheduleTranslationCreateInput!]
    }
    input RunProjectScheduleUpdateInput {
        id: ID!
        timeZone: String
        windowStart: Date
        windowEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        runProjectConnect: ID
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        translationsCreate: [RunProjectScheduleTranslationCreateInput!]
        translationsUpdate: [RunProjectScheduleTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type RunProjectSchedule {
        id: ID!
        timeZone: String
        windowStart: Date
        windowEnd: Date
        recurring: Boolean!
        recurrStart: Date
        recurrEnd: Date
        runProject: RunProject!
        labels: [Label!]!
        translations: [RunProjectScheduleTranslation!]!
    }

    input RunProjectScheduleTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input RunProjectScheduleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type RunProjectScheduleTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input RunProjectScheduleSearchInput {
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
        labelIds: [ID!]
        organizationId: ID # If not provided, uses your user ID
        searchString: String
        sortBy: RunProjectScheduleSortBy
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RunProjectScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [RunProjectScheduleEdge!]!
    }

    type RunProjectScheduleEdge {
        cursor: String!
        node: RunProjectSchedule!
    }

    extend type Query {
        runProjectSchedule(input: FindByIdInput!): RunProjectSchedule
        runProjectSchedules(input: RunProjectScheduleSearchInput!): RunProjectScheduleSearchResult!
    }

    extend type Mutation {
        runProjectScheduleCreate(input: RunProjectScheduleCreateInput!): RunProjectSchedule!
        runProjectScheduleUpdate(input: RunProjectScheduleUpdateInput!): RunProjectSchedule!
    }
`

const objectType = 'RunProjectSchedule';
export const resolvers: {
    RunProjectScheduleSortBy: typeof RunProjectScheduleSortBy;
    Query: {
        runProjectSchedule: GQLEndpoint<FindByIdInput, FindOneResult<RunProjectSchedule>>;
        runProjectSchedules: GQLEndpoint<RunProjectScheduleSearchInput, FindManyResult<RunProjectSchedule>>;
    },
    Mutation: {
        runProjectScheduleCreate: GQLEndpoint<RunProjectScheduleCreateInput, CreateOneResult<RunProjectSchedule>>;
        runProjectScheduleUpdate: GQLEndpoint<RunProjectScheduleUpdateInput, UpdateOneResult<RunProjectSchedule>>;
    }
} = {
    RunProjectScheduleSortBy,
    Query: {
        runProjectSchedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        runProjectSchedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        runProjectScheduleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runProjectScheduleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}