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
    }
    type ReminderItem {
        type: GqlModelType!
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        dueDate: Date
        completed: Boolean!
        index: Int!
        reminder: Reminder!
    }

`

export const resolvers: {
} = {
}