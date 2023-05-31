import { gql } from "apollo-server-express";

export const typeDef = gql`
    input ReminderItemCreateInput {
        id: ID!
        description: String
        dueDate: Date
        index: Int!
        name: String!
        reminderConnect: ID!
    }
    input ReminderItemUpdateInput {
        id: ID!
        description: String
        dueDate: Date
        index: Int
        isComplete: Boolean
        name: String
    }
    type ReminderItem {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        dueDate: Date
        isComplete: Boolean!
        index: Int!
        reminder: Reminder!
    }

`;

export const resolvers = {};
