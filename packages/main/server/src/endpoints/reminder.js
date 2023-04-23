import { ReminderSortBy } from "@local/consts";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
export const typeDef = gql `
    enum ReminderSortBy {
        DateCreatedAsc 
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        DueDateAsc
        DueDateDesc
        NameAsc
        NameDesc
    }

    input ReminderCreateInput {
        id: ID!
        name: String!
        description: String
        dueDate: Date
        index: Int!
        reminderListConnect: ID
        reminderListCreate: ReminderListCreateInput
        reminderItemsCreate: [ReminderItemCreateInput!]
    }
    input ReminderUpdateInput {
        id: ID!
        name: String
        description: String
        dueDate: Date
        index: Int
        isComplete: Boolean
        reminderItemsCreate: [ReminderItemCreateInput!]
        reminderItemsUpdate: [ReminderItemUpdateInput!]
        reminderItemsDelete: [ID!]
    }
    type Reminder {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        dueDate: Date
        isComplete: Boolean!
        index: Int!
        reminderList: ReminderList!
        reminderItems: [ReminderItem!]!
    }

    input ReminderSearchInput {
        ids: [ID!]
        sortBy: ReminderSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
        reminderListId: ID
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
`;
const objectType = "Reminder";
export const resolvers = {
    ReminderSortBy,
    Query: {
        reminder: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reminders: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        reminderCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reminderUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
//# sourceMappingURL=reminder.js.map