import { ChatMessageSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { ChatMessageEndpoints, EndpointsChatMessage } from "../logic/chatMessage";

export const typeDef = gql`
    enum ChatMessageSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    input ChatMessageCreateInput {
        id: ID!
        chatConnect: ID!
        versionOfId: ID
        userConnect: ID!
        translationsCreate: [ChatMessageTranslationCreateInput!]
    }
    input ChatMessageUpdateInput {
        id: ID!
        translationsDelete: [ID!]
        translationsCreate: [ChatMessageTranslationCreateInput!]
        translationsUpdate: [ChatMessageTranslationUpdateInput!]
    }

    union ChatMessageedOn = ApiVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | SmartContractVersion | StandardVersion

    type ChatMessage {
        id: ID!
        created_at: Date!
        updated_at: Date!
        sequence: Int!
        versionIndex: Int!
        parent: ChatMessageParent
        chat: Chat!
        user: User!
        reactionSummaries: [ReactionSummary!]!
        reports: [Report!]!
        reportsCount: Int!
        score: Int!
        translations: [ChatMessageTranslation!]!
        translationsCount: Int!
        you: ChatMessageYou!
    }

    type ChatMessageParent {
        id: ID!
        created_at: Date!
    }

    type ChatMessageYou {
        canDelete: Boolean!
        canUpdate: Boolean!
        canReply: Boolean!
        canReport: Boolean!
        canReact: Boolean!
        reaction: String
    }

    input ChatMessageTranslationCreateInput {
        id: ID!
        language: String!
        text: String!
    }
    input ChatMessageTranslationUpdateInput {
        id: ID!
        language: String!
        text: String
    }
    type ChatMessageTranslation {
        id: ID!
        language: String!
        text: String!
    }

    # There are two ways to search for chat messages: 
    # 1. The traditional, cursor-based pagination method. 
    # The results will not be put into a tree structure - 
    # and thus can't display version threads
    input ChatMessageSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        minScore: Int
        chatId: ID!
        userId: ID
        searchString: String
        sortBy: ChatMessageSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        translationLanguages: [String!]
    }
    type ChatMessageSearchResult {
        pageInfo: PageInfo!
        edges: [ChatMessageEdge!]!
    }
    type ChatMessageEdge {
        cursor: String!
        node: ChatMessage!
    }
    # 2. The tree-based method, which supports version threads. 
    # The best way to use this is to set the startId to the id of the last message 
    # the user has seen. With excludeUp and excludeDown both false, 
    # it will return messages above and below the startId. This way, the user doesn't 
    # lose their place.
    input ChatMessageSearchTreeInput {
        chatId: ID!
        excludeUp: Boolean
        excludeDown: Boolean
        sortBy: ChatMessageSortBy
        startId: ID
        take: Int
    }
    type ChatMessageSearchTreeResult {
        hasMoreUp: Boolean!
        hasMoreDown: Boolean!
        messages: [ChatMessage!]!
    }

    extend type Query {
        chatMessage(input: FindByIdInput!): ChatMessage
        chatMessages(input: ChatMessageSearchInput!): ChatMessageSearchResult!
        chatMessageTree(input: ChatMessageSearchTreeInput!): ChatMessageSearchTreeResult!
    }

    extend type Mutation {
        chatMessageCreate(input: ChatMessageCreateInput!): ChatMessage!
        chatMessageUpdate(input: ChatMessageUpdateInput!): ChatMessage!
    }
`;

export const resolvers: {
    ChatMessageSortBy: typeof ChatMessageSortBy;
    Query: EndpointsChatMessage["Query"];
    Mutation: EndpointsChatMessage["Mutation"];
} = {
    ChatMessageSortBy,
    ...ChatMessageEndpoints,
};
