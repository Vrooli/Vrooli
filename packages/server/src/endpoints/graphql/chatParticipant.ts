import { ChatParticipantSortBy } from "@local/shared";
import { ChatParticipantEndpoints, EndpointsChatParticipant } from "../logic/chatParticipant";

export const typeDef = `#graphql
    enum ChatParticipantSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        UserNameAsc
        UserNameDesc
    }

    input ChatParticipantUpdateInput {
        id: ID!
        #permissions: String
    }
    type ChatParticipant {
        id: ID!
        created_at: Date!
        updated_at: Date!
        #permissions: String!
        chat: Chat!
        user: User!
    }

    input ChatParticipantSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        chatId: ID
        userId: ID
        searchString: String
        sortBy: ChatParticipantSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ChatParticipantSearchResult {
        pageInfo: PageInfo!
        edges: [ChatParticipantEdge!]!
    }

    type ChatParticipantEdge {
        cursor: String!
        node: ChatParticipant!
    }

    extend type Query {
        chatParticipant(input: FindByIdInput!): ChatParticipant
        chatParticipants(input: ChatParticipantSearchInput!): ChatParticipantSearchResult!
    }

    extend type Mutation {
        chatParticipantUpdate(input: ChatParticipantUpdateInput!): ChatParticipant!
    }
`;

export const resolvers: {
    ChatParticipantSortBy: typeof ChatParticipantSortBy;
    Query: EndpointsChatParticipant["Query"];
    Mutation: EndpointsChatParticipant["Mutation"];
} = {
    ChatParticipantSortBy,
    ...ChatParticipantEndpoints,
};
