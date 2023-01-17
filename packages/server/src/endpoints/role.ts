import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Quiz, QuizCreateInput, QuizSearchInput, QuizUpdateInput, RoleSortBy } from '@shared/consts';

export const typeDef = gql`
    enum RoleSortBy {
        MembersAsc
        MembersDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input RoleCreateInput {
        id: ID!
        name: String!
        permissions: String!
        membersConnect: [ID!]
        organizationConnect: ID!
        translationsCreate: [RoleTranslationCreateInput!]
    }
    input RoleUpdateInput {
        id: ID!
        name: String
        permissions: String
        membersConnect: [ID!]
        membersDisconnect: [ID!]
        translationsDelete: [ID!]
        translationsCreate: [RoleTranslationCreateInput!]
        translationsUpdate: [RoleTranslationUpdateInput!]
    }
    type Role {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        permissions: String!
        translations: [RoleTranslation!]!
        organization: Organization!
        members: [Member!]
        membersCount: Int!
    }

    #type Permission {
    #    members: MemberPermission!
    #    organizations: OrganizationPermission!
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

    input RoleSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        languages: [String!]
        organizationId: ID!
        searchString: String
        sortBy: RoleSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }
    type RoleSearchResult {
        pageInfo: PageInfo!
        edges: [RoleEdge!]!
    }
    type RoleEdge {
        cursor: String!
        node: Role!
    }

    extend type Query {
        role(input: FindByIdInput!): Role
        roles(input: RoleSearchInput!): RoleSearchResult!
    }
    extend type Mutation {
        roleCreate(input: RoleCreateInput!): Role!
        roleUpdate(input: RoleUpdateInput!): Role!
    }
`

const objectType = 'Role';
export const resolvers: {
    RoleSortBy: typeof RoleSortBy;
    Query: {
        role: GQLEndpoint<FindByIdInput, FindOneResult<Quiz>>;
        roles: GQLEndpoint<QuizSearchInput, FindManyResult<Quiz>>;
    },
    Mutation: {
        roleCreate: GQLEndpoint<QuizCreateInput, CreateOneResult<Quiz>>;
        roleUpdate: GQLEndpoint<QuizUpdateInput, UpdateOneResult<Quiz>>;
    }
} = {
    RoleSortBy,
    Query: {
        role: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        roles: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        roleCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        roleUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}