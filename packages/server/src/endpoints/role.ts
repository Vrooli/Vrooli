import { gql } from 'apollo-server-express';

export const typeDef = gql`
    type UserRole {
        user: User!
        role: Role!
    }

    input RoleCreateInput {
        id: ID!
        name: String!
        translationsCreate: [RoleTranslationCreateInput!]
    }
    input RoleUpdateInput {
        id: ID!
        #name: String
        translationsDelete: [ID!]
        translationsCreate: [RoleTranslationCreateInput!]
        translationsUpdate: [RoleTranslationUpdateInput!]
    }
    type Role {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        translations: [RoleTranslation!]!
        organization: Organization!
        assignees: [UserRole!]
        # permissions: [Permission!]!
    }

    #type Permission {
    #    members: MemberPermission!
    #    organizations: OrganizationPermission!
    #    projects: ProjectPermission!
    #    roles: RolePermission!
    #    routines: RoutinePermission!
    #    standards: StandardPermission!
    #    users: UserPermission!
    #}

    #type MemberPermission {
    #    invite: Boolean!
    #    remove: Boolean!
    #}

    #type OrganizationPermission {
    #    create: Boolean
    #    read: Boolean
    #    update: Boolean
    #    delete: Boolean
    #}


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