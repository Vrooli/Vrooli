import { FindByIdInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

export const typeDef = gql`
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
        organizationConnect: ID
        restrictedToRolesConnect: [ID!]
        invitesCreate: [ChatInviteCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
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
        translationsCreate: [ChatTranslationCreateInput!]
        translationsUpdate: [ChatTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type Chat {
        id: ID!
        openToAnyoneWithInvite: Boolean!
        organization: Organization
        restrictedToRoles: [Role!]!
        participants: [User!]!
        participantsCount: Int!
        invites: [ChatInvite!]!
        invitesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        messages: [ChatMessage!]!
        translations: [ChatTranslation!]!
        translationsCount: Int!
        you: ChatYou!
    }

    type ChatYou {
        canDelete: Boolean!
        canInvite: Boolean!
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
        language: String
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
        ids: [ID!]
        openToAnyoneWithInvite: Boolean
        labelsIds: [ID!]
        organizationId: ID
        searchString: String
        sortBy: ChatSortBy
        take: Int
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
`

const objectType = 'Chat';
export const resolvers: {
    // ChatSortBy: typeof MeetingSortBy;
    Query: {
        chat: GQLEndpoint<FindByIdInput, FindOneResult<any>>;
        chats: GQLEndpoint<any, FindManyResult<any>>;
    },
    Mutation: {
        chatCreate: GQLEndpoint<any, CreateOneResult<any>>;
        chatUpdate: GQLEndpoint<any, UpdateOneResult<any>>;
    }
} = {
    // MeetingSortBy,
    Query: {
        chat: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        chats: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        chatCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        chatUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}