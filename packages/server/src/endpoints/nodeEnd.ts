import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeEndCreateInput {
        id: ID!
        wasSuccessful: Boolean
    }
    input NodeEndUpdateInput {
        id: ID!
        wasSuccessful: Boolean
    }
    type NodeEnd {
        id: ID!
        wasSuccessful: Boolean!
    }
`
export const resolvers: {
} = {
}