import { MeetingInviteSortBy, MeetingInviteStatus } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsMeetingInvite, MeetingInviteEndpoints } from "../logic";

export const typeDef = gql`
    enum MeetingInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        MeetingStartAsc
        MeetingStartDesc
        MeetingEndAsc
        MeetingEndDesc
        UserNameAsc
        UserNameDesc
    }

    enum MeetingInviteStatus {
        Pending
        Accepted
        Declined
    }

    input MeetingInviteCreateInput {
        id: ID!
        meetingConnect: ID!
        userConnect: ID!
        message: String
    }
    input MeetingInviteUpdateInput {
        id: ID!
        message: String
    }
    type MeetingInvite {
        id: ID!
        created_at: Date!
        updated_at: Date!
        meeting: Meeting!
        user: User!
        message: String
        status: MeetingInviteStatus!
        you: MeetingInviteYou!
    }

    type MeetingInviteYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input MeetingInviteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: MeetingInviteStatus
        meetingId: ID
        userId: ID
        organizationId: ID
        searchString: String
        sortBy: MeetingInviteSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MeetingInviteSearchResult {
        pageInfo: PageInfo!
        edges: [MeetingInviteEdge!]!
    }

    type MeetingInviteEdge {
        cursor: String!
        node: MeetingInvite!
    }

    extend type Query {
        meetingInvite(input: FindByIdInput!): MeetingInvite
        meetingInvites(input: MeetingInviteSearchInput!): MeetingInviteSearchResult!
    }

    extend type Mutation {
        meetingInviteCreate(input: MeetingInviteCreateInput!): MeetingInvite!
        meetingInviteUpdate(input: MeetingInviteUpdateInput!): MeetingInvite!
        meetingInviteAccept(input: FindByIdInput!): MeetingInvite!
        meetingInviteDecline(input: FindByIdInput!): MeetingInvite!
    }
`;

export const resolvers: {
    MeetingInviteSortBy: typeof MeetingInviteSortBy;
    MeetingInviteStatus: typeof MeetingInviteStatus;
    Query: EndpointsMeetingInvite["Query"];
    Mutation: EndpointsMeetingInvite["Mutation"];
} = {
    MeetingInviteSortBy,
    MeetingInviteStatus,
    ...MeetingInviteEndpoints,
};
