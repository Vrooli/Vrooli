import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type Payment {
        id: ID!
    }
`

export const resolvers = {}