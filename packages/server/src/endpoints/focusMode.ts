import { FindByIdInput, FocusMode, FocusModeCreateInput, FocusModeSearchInput, FocusModeSortBy, FocusModeStopCondition, FocusModeUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

export const typeDef = gql`
    enum FocusModeSortBy {
        NameAsc
        NameDesc
        EventStartAsc
        EventStartDesc
        EventEndAsc
        EventEndDesc
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
    }

    enum FocusModeStopCondition {
        AfterStopTime
        Automatic
        Never
        NextBegins
    }

    input FocusModeCreateInput {
        id: ID!
        name: String!
        description: String
        filtersCreate: [FocusModeFilterCreateInput!]
        reminderListConnect: ID
        reminderListCreate: ReminderListCreateInput
        resourceListCreate: ResourceListCreateInput
        scheduleCreate: ScheduleCreateInput
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    input FocusModeUpdateInput {
        id: ID!
        name: String
        description: String
        filtersCreate: [FocusModeFilterCreateInput!]
        filtersDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        reminderListConnect: ID
        reminderListDisconnect: Boolean
        reminderListCreate: ReminderListCreateInput
        reminderListUpdate: ReminderListUpdateInput
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        scheduleCreate: ScheduleCreateInput
        scheduleUpdate: ScheduleUpdateInput
    }
    type FocusMode {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        filters: [FocusModeFilter!]!
        labels: [Label!]!
        reminderList: ReminderList
        resourceList: ResourceList
        schedule: Schedule
    }

    type ActiveFocusMode {
        mode: FocusMode!
        stopCondition: FocusModeStopCondition!
        stopTime: Date    
    }

    input FocusModeSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        eventStartTimeFrame: TimeFrame
        eventEndTimeFrame: TimeFrame
        ids: [ID!]
        recurrStartTimeFrame: TimeFrame
        recurrEndTimeFrame: TimeFrame
        searchString: String
        sortBy: FocusModeSortBy
        labelsIds: [ID!]
        take: Int
        timeZone: String
        updatedTimeFrame: TimeFrame
    }

    type FocusModeSearchResult {
        pageInfo: PageInfo!
        edges: [FocusModeEdge!]!
    }

    type FocusModeEdge {
        cursor: String!
        node: FocusMode!
    }

    extend type Query {
        focusMode(input: FindByIdInput!): FocusMode
        focusModes(input: FocusModeSearchInput!): FocusModeSearchResult!
    }

    extend type Mutation {
        focusModeCreate(input: FocusModeCreateInput!): FocusMode!
        focusModeUpdate(input: FocusModeUpdateInput!): FocusMode!
    }
`

const objectType = 'FocusMode';
export const resolvers: {
    FocusModeSortBy: typeof FocusModeSortBy;
    FocusModeStopCondition: typeof FocusModeStopCondition;
    Query: {
        focusMode: GQLEndpoint<FindByIdInput, FindOneResult<FocusMode>>;
        focusModes: GQLEndpoint<FocusModeSearchInput, FindManyResult<FocusMode>>;
    },
    Mutation: {
        focusModeCreate: GQLEndpoint<FocusModeCreateInput, CreateOneResult<FocusMode>>;
        focusModeUpdate: GQLEndpoint<FocusModeUpdateInput, UpdateOneResult<FocusMode>>;
    }
} = {
    FocusModeSortBy,
    FocusModeStopCondition,
    Query: {
        focusMode: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        focusModes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        focusModeCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        focusModeUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}