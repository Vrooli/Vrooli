import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input ReminderItemCreateInput {
        id: ID!
        reminderId: ID!
        index: Int
        link: String!
    }
    input ReminderItemUpdateInput {
        id: ID!
        index: Int
        link: String
    }
    type ReminderItem {
        id: ID!
        created_at: Date!
        updated_at: Date!
        listId: ID!
        index: Int
        link: String!
    }

`

export const resolvers: {
} = {
}