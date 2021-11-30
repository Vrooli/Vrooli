import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type CustomerRole {
        customer: Customer!
        role: Role!
    }
    type Role {
        id: ID!
        title: String!
        description: String
        customers: [Customer!]!
    }
`

export const resolvers = {}