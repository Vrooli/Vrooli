import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, ReminderSortBy, Reminder, ReminderSearchInput, ReminderCreateInput, ReminderUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ReminderSortBy {
        NameAsc
        NameDesc
        DueDateAsc
        DueDateDesc
    }

    input ReminderCreateInput {
        id: ID!
        listId: ID!
        index: Int
        link: String!
        translationsCreate: [ReminderTranslationCreateInput!]
    }
    input ReminderUpdateInput {
        id: ID!
        listId: ID
        index: Int
        link: String
        translationsDelete: [ID!]
        translationsCreate: [ReminderTranslationCreateInput!]
        translationsUpdate: [ReminderTranslationUpdateInput!]
    }
    type Reminder {
        id: ID!
        created_at: Date!
        updated_at: Date!
        listId: ID!
        index: Int
        link: String!
        translations: [ReminderTranslation!]!
    }

    input ReminderTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input ReminderTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ReminderTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }

    input ReminderSearchInput {
        forId: ID
        ids: [ID!]
        languages: [String!]
        sortBy: ReminderSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    type ReminderSearchResult {
        pageInfo: PageInfo!
        edges: [ReminderEdge!]!
    }

    type ReminderEdge {
        cursor: String!
        node: Reminder!
    }

    extend type Query {
        reminder(input: FindByIdInput!): Reminder
        reminders(input: ReminderSearchInput!): ReminderSearchResult!
    }

    extend type Mutation {
        reminderCreate(input: ReminderCreateInput!): Reminder!
        reminderUpdate(input: ReminderUpdateInput!): Reminder!
    }
`

const objectType = 'Reminder';
export const resolvers: {
    ReminderSortBy: typeof ReminderSortBy;
    Query: {
        reminder: GQLEndpoint<FindByIdInput, FindOneResult<Reminder>>;
        reminders: GQLEndpoint<ReminderSearchInput, FindManyResult<Reminder>>;
    },
    Mutation: {
        reminderCreate: GQLEndpoint<ReminderCreateInput, CreateOneResult<Reminder>>;
        reminderUpdate: GQLEndpoint<ReminderUpdateInput, UpdateOneResult<Reminder>>;
    }
} = {
    ReminderSortBy,
    Query: {
        reminder: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        reminders: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        reminderCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        reminderUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}