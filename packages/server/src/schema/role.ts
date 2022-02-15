import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type UserRole {
        user: User!
        role: Role!
    }
    type Role {
        id: ID!
        title: String!
        description: String
        users: [User!]!
    }
`

export const resolvers = {}