import { QuizQuestionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsQuizQuestion, QuizQuestionEndpoints } from "../logic/quizQuestion";

export const typeDef = gql`
    enum QuizQuestionSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        OrderAsc
        OrderDesc
    }

    input QuizQuestionCreateInput {
        id: ID!
        order: Int
        points: Int
        standardVersionConnect: ID
        standardVersionCreate: StandardVersionCreateInput
        quizConnect: ID!
        translationsCreate: [QuizQuestionTranslationCreateInput!]
    }
    input QuizQuestionUpdateInput {
        id: ID!
        order: Int
        points: Int
        standardVersionConnect: ID
        standardVersionCreate: StandardVersionCreateInput
        standardVersionUpdate: StandardVersionUpdateInput
        translationsCreate: [QuizQuestionTranslationCreateInput!]
        translationsUpdate: [QuizQuestionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type QuizQuestion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        order: Int
        points: Int!
        quiz: Quiz!
        responses: [QuizQuestionResponse!]
        responsesCount: Int!
        standardVersion: StandardVersion
        translations: [QuizQuestionTranslation!]
        you: QuizQuestionYou!
    }

    type QuizQuestionYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input QuizQuestionTranslationCreateInput {
        id: ID!
        language: String!
        helpText: String
        questionText: String!
    }
    input QuizQuestionTranslationUpdateInput {
        id: ID!
        language: String!
        helpText: String
        questionText: String
    }
    type QuizQuestionTranslation {
        id: ID!
        language: String!
        helpText: String
        questionText: String!
    }

    input QuizQuestionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        translationLanguages: [String!]
        quizId: ID
        standardId: ID
        userId: ID
        responseId: ID
        searchString: String
        sortBy: QuizQuestionSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }
    type QuizQuestionSearchResult {
        pageInfo: PageInfo!
        edges: [QuizQuestionEdge!]!
    }
    type QuizQuestionEdge {
        cursor: String!
        node: QuizQuestion!
    }

    extend type Query {
        quizQuestion(input: FindByIdInput!): QuizQuestion
        quizQuestions(input: QuizQuestionSearchInput!): QuizQuestionSearchResult!
    }
`;

export const resolvers: {
    QuizQuestionSortBy: typeof QuizQuestionSortBy;
    Query: EndpointsQuizQuestion["Query"];
} = {
    QuizQuestionSortBy,
    ...QuizQuestionEndpoints,
};
