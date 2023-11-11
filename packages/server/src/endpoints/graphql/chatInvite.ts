import { ChatInviteSortBy, ChatInviteStatus } from "@local/shared";
import { gql } from "apollo-server-express";
import { ChatInviteEndpoints, EndpointsChatInvite } from "../logic/chatInvite";

export const typeDef = gql`
    enum ChatInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        UserNameAsc
        UserNameDesc
    }

    enum ChatInviteStatus {
        Pending
        Accepted
        Declined
    }

    input ChatInviteCreateInput {
        id: ID!
        userConnect: ID!
        chatConnect: ID!
        message: String
        #willHavePermissions: String
    }
    input ChatInviteUpdateInput {
        id: ID!
        message: String
        #willHavePermissions: String
    }
    type ChatInvite {
        id: ID!
        created_at: Date!
        updated_at: Date!
        user: User!
        chat: Chat!
        message: String
        status: ChatInviteStatus!
        #willHavePermissions: String
        you: ChatInviteYou!
    }

    type ChatInviteYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input ChatInviteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: ChatInviteStatus
        statuses: [ChatInviteStatus!]
        chatId: ID
        userId: ID
        searchString: String
        sortBy: ChatInviteSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ChatInviteSearchResult {
        pageInfo: PageInfo!
        edges: [ChatInviteEdge!]!
    }

    type ChatInviteEdge {
        cursor: String!
        node: ChatInvite!
    }

    extend type Query {
        chatInvite(input: FindByIdInput!): ChatInvite
        chatInvites(input: ChatInviteSearchInput!): ChatInviteSearchResult!
    }

    extend type Mutation {
        chatInviteCreate(input: ChatInviteCreateInput!): ChatInvite!
        chatInvitesCreate(input: [ChatInviteCreateInput!]!): [ChatInvite!]!
        chatInviteUpdate(input: ChatInviteUpdateInput!): ChatInvite!
        chatInvitesUpdate(input: [ChatInviteUpdateInput!]!): [ChatInvite!]!
        chatInviteAccept(input: FindByIdInput!): ChatInvite!
        chatInviteDecline(input: FindByIdInput!): ChatInvite!
    }
`;

export const resolvers: {
    ChatInviteSortBy: typeof ChatInviteSortBy;
    ChatInviteStatus: typeof ChatInviteStatus;
    Query: EndpointsChatInvite["Query"];
    Mutation: EndpointsChatInvite["Mutation"];
} = {
    ChatInviteSortBy,
    ChatInviteStatus,
    ...ChatInviteEndpoints,
};
