import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, ReminderList, ReminderListSearchInput, ReminderListCreateInput, ReminderListUpdateInput, ReminderListSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ReminderListSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input ReminderListCreateInput {
        id: ID!
        index: Int
        organizationId: ID
        projectId: ID
        routineId: ID
        userId: ID
        translationsCreate: [ReminderListTranslationCreateInput!]
        resourcesCreate: [ResourceCreateInput!]
    }
    input ReminderListUpdateInput {
        id: ID!
        index: Int
        organizationId: ID
        projectId: ID
        routineId: ID
        userId: ID
        translationsDelete: [ID!]
        translationsCreate: [ReminderListTranslationCreateInput!]
        translationsUpdate: [ReminderListTranslationUpdateInput!]
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
    }
    type ReminderList {
        id: ID!
        created_at: Date!
        updated_at: Date!
        index: Int
        # api: Api
        organization: Organization
        # post: Post
        project: Project
        routine: Routine
        # smartContract: SmartContract
        standard: Standard
        # userSchedule: UserSchedule
        translations: [ReminderListTranslation!]!
        resources: [Resource!]!
    }

    input ReminderListTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input ReminderListTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ReminderListTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }

    input ReminderListSearchInput {
        organizationId: ID
        projectId: ID
        routineId: ID
        userId: ID
        ids: [ID!]
        languages: [String!]
        sortBy: ReminderListSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    type ReminderListSearchResult {
        pageInfo: PageInfo!
        edges: [ReminderListEdge!]!
    }

    type ReminderListEdge {
        cursor: String!
        node: ReminderList!
    }

    extend type Query {
        reminderList(input: FindByIdInput!): ReminderList
        reminderLists(input: ReminderListSearchInput!): ReminderListSearchResult!
    }

    extend type Mutation {
        reminderListCreate(input: ReminderListCreateInput!): ReminderList!
        reminderListUpdate(input: ReminderListUpdateInput!): ReminderList!
    }
`

const objectType = 'ReminderList';
export const resolvers: {
    ReminderListSortBy: typeof ReminderListSortBy;
    Query: {
        reminderList: GQLEndpoint<FindByIdInput, FindOneResult<ReminderList>>;
        reminderLists: GQLEndpoint<ReminderListSearchInput, FindManyResult<ReminderList>>;
    },
    Mutation: {
        reminderListCreate: GQLEndpoint<ReminderListCreateInput, CreateOneResult<ReminderList>>;
        reminderListUpdate: GQLEndpoint<ReminderListUpdateInput, UpdateOneResult<ReminderList>>;
    }
} = {
    ReminderListSortBy,
    Query: {
        reminderList: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reminderLists: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        reminderListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reminderListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}