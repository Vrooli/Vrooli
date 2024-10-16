import { MemberInviteSortBy, MemberInviteStatus } from "@local/shared";
import { EndpointsMemberInvite, MemberInviteEndpoints } from "../logic/memberInvite";

export const typeDef = `#graphql
    enum MemberInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        UserNameAsc
        UserNameDesc
    }

    enum MemberInviteStatus {
        Pending
        Accepted
        Declined
    }

    input MemberInviteCreateInput {
        id: ID!
        message: String
        teamConnect: ID!
        userConnect: ID!
        willBeAdmin: Boolean
        willHavePermissions: String
    }
    input MemberInviteUpdateInput {
        id: ID!
        message: String
        willBeAdmin: Boolean
        willHavePermissions: String
    }
    type MemberInvite {
        id: ID!
        created_at: Date!
        updated_at: Date!
        user: User!
        team: Team!
        message: String
        status: MemberInviteStatus!
        willBeAdmin: Boolean!
        willHavePermissions: String
        you: MemberInviteYou!
    }

    type MemberInviteYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input MemberInviteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: MemberInviteStatus
        statuses: [MemberInviteStatus!]
        teamId: ID
        userId: ID
        searchString: String
        sortBy: MemberInviteSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MemberInviteSearchResult {
        pageInfo: PageInfo!
        edges: [MemberInviteEdge!]!
    }

    type MemberInviteEdge {
        cursor: String!
        node: MemberInvite!
    }

    extend type Query {
        memberInvite(input: FindByIdInput!): MemberInvite
        memberInvites(input: MemberInviteSearchInput!): MemberInviteSearchResult!
    }

    extend type Mutation {
        memberInviteCreate(input: MemberInviteCreateInput!): MemberInvite!
        memberInvitesCreate(input: [MemberInviteCreateInput!]!): [MemberInvite!]!
        memberInviteUpdate(input: MemberInviteUpdateInput!): MemberInvite!
        memberInvitesUpdate(input: [MemberInviteUpdateInput!]!): [MemberInvite!]!
        memberInviteAccept(input: FindByIdInput!): MemberInvite!
        memberInviteDecline(input: FindByIdInput!): MemberInvite!
    }
`;

export const resolvers: {
    MemberInviteSortBy: typeof MemberInviteSortBy;
    MemberInviteStatus: typeof MemberInviteStatus;
    Query: EndpointsMemberInvite["Query"];
    Mutation: EndpointsMemberInvite["Mutation"];
} = {
    MemberInviteSortBy,
    MemberInviteStatus,
    ...MemberInviteEndpoints,
};
