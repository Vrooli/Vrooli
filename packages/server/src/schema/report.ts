import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type Report {
        id: ID!
        created_at: Date!
        reason: String!
        details: String
        fromId: ID!
        from: User!
    }
`

export const resolvers = {}