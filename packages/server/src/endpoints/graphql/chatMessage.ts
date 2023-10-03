import { ChatMessageSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { ChatMessageEndpoints, EndpointsChatMessage } from "../logic";

export const typeDef = gql`
    enum ChatMessageSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ScoreAsc
        ScoreDesc
    }

    input ChatMessageCreateInput {
        id: ID!
        chatConnect: ID!
        isFork: Boolean!
        forkId: ID
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
        isFork: Boolean!
        fork: ChatMessageFork
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

    type ChatMessageFork {
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

    input ChatMessageSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        minScore: Int
        chatId: ID
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

    extend type Query {
        chatMessage(input: FindByIdInput!): ChatMessage
        chatMessages(input: ChatMessageSearchInput!): ChatMessageSearchResult!
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
