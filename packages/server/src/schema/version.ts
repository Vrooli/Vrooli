import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input Version {
        id: ID!
        label: String!
        notes: String
    }
`

export const resolvers = {
}