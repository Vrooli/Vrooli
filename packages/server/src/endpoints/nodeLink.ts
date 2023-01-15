import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeLinkCreateInput {
        id: ID!
        operation: String
        fromConnect: ID!
        toConnect: ID!
        routineVersionConnect: ID!
        whensCreate: [NodeLinkWhenCreateInput!]
    }
    input NodeLinkUpdateInput {
        id: ID!
        whensCreate: [NodeLinkWhenCreateInput!]
        whensUpdate: [NodeLinkWhenUpdateInput!]
        whensDelete: [ID!]
        operation: String
        fromConnect: ID
        fromDisconnect: Boolean
        toConnect: ID
        toDisconnect: Boolean
    }
    type NodeLink {
        id: ID!
        whens: [NodeLinkWhen!]!
        operation: String
        from: Node!
        to: Node!
        routineVersion: RoutineVersion!
    }
`
export const resolvers: {
} = {
}