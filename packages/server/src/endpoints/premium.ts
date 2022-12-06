import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type Premium {
        id: ID!
    }
`

export const resolvers = {}