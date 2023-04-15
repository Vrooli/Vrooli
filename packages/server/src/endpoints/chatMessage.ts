import { FindByIdInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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
        chat: Chat!
        user: User!
        reports: [Report!]!
        reportsCount: Int!
        score: Int!
        translations: [ChatMessageTranslation!]!
        translationsCount: Int!
        you: ChatMessageYou!
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
        language: String
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
`

const objectType = 'ChatMessage';
export const resolvers: {
    // ChatMessageFor: typeof ChatMessageFor;
    // ChatMessageSortBy: typeof ChatMessageSortBy;
    Query: {
        chatMessage: GQLEndpoint<FindByIdInput, FindOneResult<any>>;
        chatMessages: GQLEndpoint<any, any>;
    },
    Mutation: {
        chatMessageCreate: GQLEndpoint<any, CreateOneResult<any>>;
        chatMessageUpdate: GQLEndpoint<any, UpdateOneResult<any>>;
    }
} = {
    // ChatMessageFor,
    // ChatMessageSortBy,
    Query: {
        chatMessage: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        chatMessages: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        chatMessageCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        chatMessageUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}