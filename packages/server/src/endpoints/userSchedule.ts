import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, UserScheduleSortBy, UserSchedule, UserScheduleSearchInput, UserScheduleCreateInput, UserScheduleUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum UserScheduleSortBy {
        TitleAsc
        TitleDesc
        EventStartAsc
        EventStartDesc
        EventEndAsc
        EventEndDesc
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
    }

    input UserScheduleCreateInput {
        id: ID!
        name: String!
        description: String
        timeZone: String
        eventStart: Date
        eventEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        reminderListConnect: ID
        reminderListCreate: ReminderListCreateInput
        resourceListCreate: ResourceListCreateInput
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        filtersCreate: [UserScheduleFilterCreateInput!]
    }
    input UserScheduleUpdateInput {
        id: ID!
        name: String
        description: String
        timeZone: String
        eventStart: Date
        eventEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        reminderListConnect: ID
        reminderListDisconnect: ID
        reminderListCreate: ReminderListCreateInput
        reminderListUpdate: ReminderListUpdateInput
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        filtersCreate: [UserScheduleFilterCreateInput!]
        filtersDelete: [ID!]
    }
    type UserSchedule {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        timeZone: String
        eventStart: Date
        eventEnd: Date
        recurring: Boolean!
        recurrStart: Date
        recurrEnd: Date
        reminderList: ReminderList
        labels: [Label!]!
        filters: [UserScheduleFilter!]!
    }

    input UserScheduleSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        eventStartTimeFrame: TimeFrame
        eventEndTimeFrame: TimeFrame
        ids: [ID!]
        recurrStartTimeFrame: TimeFrame
        recurrEndTimeFrame: TimeFrame
        searchString: String
        sortBy: UserScheduleSortBy
        tags: [String!]
        take: Int
        timeZone: String
        updatedTimeFrame: TimeFrame
    }

    type UserScheduleSearchResult {
        pageInfo: PageInfo!
        edges: [UserScheduleEdge!]!
    }

    type UserScheduleEdge {
        cursor: String!
        node: UserSchedule!
    }

    extend type Query {
        userSchedule(input: FindByIdInput!): UserSchedule
        userSchedules(input: UserScheduleSearchInput!): UserScheduleSearchResult!
    }

    extend type Mutation {
        userScheduleCreate(input: UserScheduleCreateInput!): UserSchedule!
        userScheduleUpdate(input: UserScheduleUpdateInput!): UserSchedule!
    }
`

const objectType = 'UserSchedule';
export const resolvers: {
    UserScheduleSortBy: typeof UserScheduleSortBy;
    Query: {
        userSchedule: GQLEndpoint<FindByIdInput, FindOneResult<UserSchedule>>;
        userSchedules: GQLEndpoint<UserScheduleSearchInput, FindManyResult<UserSchedule>>;
    },
    Mutation: {
        userScheduleCreate: GQLEndpoint<UserScheduleCreateInput, CreateOneResult<UserSchedule>>;
        userScheduleUpdate: GQLEndpoint<UserScheduleUpdateInput, UpdateOneResult<UserSchedule>>;
    }
} = {
    UserScheduleSortBy,
    Query: {
        userSchedule: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        userSchedules: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        userScheduleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        userScheduleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}