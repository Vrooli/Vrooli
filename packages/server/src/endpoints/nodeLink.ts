import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeLinkCreateInput {
        id: ID!
        whens: [NodeLinkWhenCreateInput!]
        operation: String
        fromId: ID!
        toId: ID!
    }
    input NodeLinkUpdateInput {
        id: ID!
        whensCreate: [NodeLinkWhenCreateInput!]
        whensUpdate: [NodeLinkWhenUpdateInput!]
        whensDelete: [ID!]
        operation: String
        fromId: ID
        toId: ID
    }
    type NodeLink{
        id: ID!
        whens: [NodeLinkWhen!]!
        operation: String
        fromId: ID!
        toId: ID!
    }
`
export const resolvers: {
} = {
}