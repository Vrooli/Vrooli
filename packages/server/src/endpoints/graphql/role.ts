import { RoleSortBy } from "@local/shared";
import { EndpointsRole, RoleEndpoints } from "../logic/role";

export const typeDef = `#graphql
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
        teamConnect: ID!
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
        members: [Member!]
        membersCount: Int!
        name: String!
        permissions: String!
        team: Team!
        translations: [RoleTranslation!]!
    }

    #type Permission {
    #    members: MemberPermission!
    #    teams: TeamPermission!
    #    roles: RolePermission!
    #    routines: RoutinePermission!
    #    standards: StandardPermission!
    #    users: UserPermission!
    #}

    #type MemberPermission {
    #    invite: Boolean!
    #    remove: Boolean!
    #}

    #type TeamPermission {
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
        language: String!
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
        teamId: ID!
        translationLanguages: [String!]
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
`;

export const resolvers: {
    RoleSortBy: typeof RoleSortBy;
    Query: EndpointsRole["Query"];
    Mutation: EndpointsRole["Mutation"];
} = {
    RoleSortBy,
    ...RoleEndpoints,
};
