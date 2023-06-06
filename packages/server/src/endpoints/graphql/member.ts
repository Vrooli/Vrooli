import { MemberSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsMember, MemberEndpoints } from "../logic";

export const typeDef = gql`
    enum MemberSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input MemberUpdateInput {
        id: ID!
        isAdmin: Boolean
        permissions: String
    }
    type Member {
        id: ID!
        created_at: Date!
        updated_at: Date!
        isAdmin: Boolean!
        permissions: String!
        organization: Organization!
        roles: [Role!]!
        user: User!
    }

    input MemberSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isAdmin: Boolean
        roles: [String!]
        organizationId: ID
        userId: ID
        searchString: String
        sortBy: MemberSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MemberSearchResult {
        pageInfo: PageInfo!
        edges: [MemberEdge!]!
    }

    type MemberEdge {
        cursor: String!
        node: Member!
    }

    extend type Query {
        member(input: FindByIdInput!): Member
        members(input: MemberSearchInput!): MemberSearchResult!
    }

    extend type Mutation {
        memberUpdate(input: MemberUpdateInput!): Member!
    }
`;

export const resolvers: {
    MemberSortBy: typeof MemberSortBy;
    Query: EndpointsMember["Query"];
    Mutation: EndpointsMember["Mutation"];
} = {
    MemberSortBy,
    ...MemberEndpoints,
};
