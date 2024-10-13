import { ChatSortBy } from "@local/shared";
import { ChatEndpoints, EndpointsChat } from "../logic/chat";

export const typeDef = `#graphql
    enum ChatSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ParticipantsAsc
        ParticipantsDesc
        InvitesAsc
        InvitesDesc
        MessagesAsc
        MessagesDesc
    }

    input ChatCreateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        task: String # If chatting with Valyxa or another bot, this is used to set up the initial message
        restrictedToRolesConnect: [ID!]
        invitesCreate: [ChatInviteCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        messagesCreate: [ChatMessageCreateInput!]
        teamConnect: ID
        translationsCreate: [ChatTranslationCreateInput!]
    }
    input ChatUpdateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        restrictedToRolesConnect: [ID!]
        restrictedToRolesDisconnect: [ID!]
        invitesCreate: [ChatInviteCreateInput!]
        invitesUpdate: [ChatInviteUpdateInput!]
        invitesDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        messagesCreate: [ChatMessageCreateInput!]
        messagesUpdate: [ChatMessageUpdateInput!]
        messagesDelete: [ID!]
        participantsDelete: [ID!]
        translationsCreate: [ChatTranslationCreateInput!]
        translationsUpdate: [ChatTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type Chat {
        id: ID!
        created_at: Date!
        updated_at: Date!
        openToAnyoneWithInvite: Boolean!
        restrictedToRoles: [Role!]!
        participants: [ChatParticipant!]!
        participantsCount: Int!
        invites: [ChatInvite!]!
        invitesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        messages: [ChatMessage!]!
        team: Team
        translations: [ChatTranslation!]!
        translationsCount: Int!
        you: ChatYou!
    }

    type ChatYou {
        canDelete: Boolean!
        canInvite: Boolean!
        canRead: Boolean!
        canUpdate: Boolean!
    }

    input ChatTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input ChatTranslationUpdateInput {
        id: ID!
        language: String!
        name: String
        description: String
    }
    type ChatTranslation {
        id: ID!
        language: String!
        name: String
        description: String
    }

    input ChatSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        creatorId: ID
        ids: [ID!]
        openToAnyoneWithInvite: Boolean
        labelsIds: [ID!]
        searchString: String
        sortBy: ChatSortBy
        take: Int
        teamId: ID
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ChatSearchResult {
        pageInfo: PageInfo!
        edges: [ChatEdge!]!
    }

    type ChatEdge {
        cursor: String!
        node: Chat!
    }

    extend type Query {
        chat(input: FindByIdInput!): Chat
        chats(input: ChatSearchInput!): ChatSearchResult!
    }

    extend type Mutation {
        chatCreate(input: ChatCreateInput!): Chat!
        chatUpdate(input: ChatUpdateInput!): Chat!
    }
`;

export const resolvers: {
    ChatSortBy: typeof ChatSortBy;
    Query: EndpointsChat["Query"];
    Mutation: EndpointsChat["Mutation"];
} = {
    ChatSortBy,
    ...ChatEndpoints,
};
