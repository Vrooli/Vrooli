import { ReminderSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsReminder, ReminderEndpoints } from "../logic";

export const typeDef = gql`
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

export const resolvers: {
    ReminderSortBy: typeof ReminderSortBy;
    Query: EndpointsReminder["Query"];
    Mutation: EndpointsReminder["Mutation"];
} = {
    ReminderSortBy,
    ...ReminderEndpoints,
};
