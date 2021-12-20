import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type Report {
        id: ID!
        reason: String!
        details: String
        created_at: Date!
        fromId: ID!
        from: User!
    }
`

export const resolvers = {}