import { FindByIdInput, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionSearchInput, ScheduleExceptionSortBy, ScheduleExceptionUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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
`

const objectType = 'ScheduleException';
export const resolvers: {
    ScheduleExceptionSortBy: typeof ScheduleExceptionSortBy;
    Query: {
        scheduleException: GQLEndpoint<FindByIdInput, FindOneResult<ScheduleException>>;
        scheduleExceptions: GQLEndpoint<ScheduleExceptionSearchInput, FindManyResult<ScheduleException>>;
    },
    Mutation: {
        scheduleExceptionCreate: GQLEndpoint<ScheduleExceptionCreateInput, CreateOneResult<ScheduleException>>;
        scheduleExceptionUpdate: GQLEndpoint<ScheduleExceptionUpdateInput, UpdateOneResult<ScheduleException>>;
    }
} = {
    ScheduleExceptionSortBy,
    Query: {
        scheduleException: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        scheduleExceptions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        scheduleExceptionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        scheduleExceptionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}