import { QuestionAnswerSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsQuestionAnswer, QuestionAnswerEndpoints } from "../logic/questionAnswer";

export const typeDef = gql`
    enum QuestionAnswerSortBy {
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
    }

    input QuestionAnswerCreateInput {
        id: ID!
        questionConnect: ID!
        translationsCreate: [QuestionAnswerTranslationCreateInput!]
    }
    input QuestionAnswerUpdateInput {
        id: ID!
        translationsDelete: [ID!]
        translationsCreate: [QuestionAnswerTranslationCreateInput!]
        translationsUpdate: [QuestionAnswerTranslationUpdateInput!]
    }
    type QuestionAnswer {
        id: ID!
        created_at: Date!
        updated_at: Date!
        createdBy: User
        bookmarks: Int!
        score: Int!
        isAccepted: Boolean!
        question: Question!
        comments: [Comment!]!
        commentsCount: Int!
        bookmarkedBy: [User!]!
        translations: [QuestionAnswerTranslation!]!
    }

    input QuestionAnswerTranslationCreateInput {
        id: ID!
        language: String!
        text: String!
    }
    input QuestionAnswerTranslationUpdateInput {
        id: ID!
        language: String!
        text: String
    }
    type QuestionAnswerTranslation {
        id: ID!
        language: String!
        text: String!
    }

    input QuestionAnswerSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        translationLanguages: [String!]
        minScore: Int
        minBookmarks: Int
        searchString: String
        sortBy: QuestionAnswerSortBy
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type QuestionAnswerSearchResult {
        pageInfo: PageInfo!
        edges: [QuestionAnswerEdge!]!
    }

    type QuestionAnswerEdge {
        cursor: String!
        node: QuestionAnswer!
    }

    extend type Query {
        questionAnswer(input: FindByIdInput!): QuestionAnswer
        questionAnswers(input: QuestionAnswerSearchInput!): QuestionAnswerSearchResult!
    }

    extend type Mutation {
        questionAnswerCreate(input: QuestionAnswerCreateInput!): QuestionAnswer!
        questionAnswerUpdate(input: QuestionAnswerUpdateInput!): QuestionAnswer!
        questionAnswerMarkAsAccepted(input: FindByIdInput!): QuestionAnswer!
    }
`;

export const resolvers: {
    QuestionAnswerSortBy: typeof QuestionAnswerSortBy;
    Query: EndpointsQuestionAnswer["Query"];
    Mutation: EndpointsQuestionAnswer["Mutation"];
} = {
    QuestionAnswerSortBy,
    ...QuestionAnswerEndpoints,
};
