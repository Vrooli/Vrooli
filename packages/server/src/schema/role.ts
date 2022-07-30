import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type UserRole {
        user: User!
        role: Role!
    }

    input RoleCreateInput {
        id: ID!
        title: String!
        translationsCreate: [RoleTranslationCreateInput!]
    }
    input RoleUpdateInput {
        id: ID!
        #title: String
        translationsDelete: [ID!]
        translationsCreate: [RoleTranslationCreateInput!]
        translationsUpdate: [RoleTranslationUpdateInput!]
    }
    type Role {
        id: ID!
        created_at: Date!
        updated_at: Date!
        title: String!
        translations: [RoleTranslation!]!
        organization: Organization!
        assignees: [UserRole!]
    }

    input RoleTranslationCreateInput {
        id: ID!
        language: String!
        description: String!
    }

    input RoleTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }

    type RoleTranslation {
        id: ID!
        language: String!
        description: String!
    }
`

export const resolvers = {}