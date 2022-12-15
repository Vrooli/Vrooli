import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeEndCreateInput {
        id: ID!
        wasSuccessful: Boolean
        suggestedNextRoutineVersionConnect: ID
    }
    input NodeEndUpdateInput {
        id: ID!
        wasSuccessful: Boolean
        suggestedNextRoutineVersionConnect: ID
        suggestedNextRoutineVersionDisconnect: ID
    }
    type NodeEnd {
        id: ID!
        wasSuccessful: Boolean!
        suggestedNextRoutineVersion: RoutineVersion
    }
`
export const resolvers: {
} = {
}