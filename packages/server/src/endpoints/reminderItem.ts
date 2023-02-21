import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input ReminderItemCreateInput {
        id: ID!
        name: String!
        description: String
        dueDate: Date
        index: Int!
    }
    input ReminderItemUpdateInput {
        id: ID!
        name: String
        description: String
        dueDate: Date
        index: Int
        isComplete: Boolean
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

`

export const resolvers: {
} = {
}