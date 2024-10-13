import { QuizQuestionResponseSortBy } from "@local/shared";
import { EndpointsQuizQuestionResponse, QuizQuestionResponseEndpoints } from "../logic/quizQuestionResponse";

export const typeDef = `#graphql
    enum QuizQuestionResponseSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        QuestionOrderAsc
        QuestionOrderDesc
    }

    input QuizQuestionResponseCreateInput {
        id: ID!
        response: String!
        quizAttemptConnect: ID!
        quizQuestionConnect: ID!
    }
    input QuizQuestionResponseUpdateInput {
        id: ID!
        response: String
    }
    type QuizQuestionResponse {
        id: ID!
        created_at: Date!
        updated_at: Date!
        response: String
        quizAttempt: QuizAttempt!
        quizQuestion: QuizQuestion!
        you: QuizQuestionResponseYou!
    }

    type QuizQuestionResponseYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input QuizQuestionResponseSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        quizAttemptId: ID
        quizQuestionId: ID
        searchString: String
        sortBy: QuizQuestionResponseSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type QuizQuestionResponseSearchResult {
        pageInfo: PageInfo!
        edges: [QuizQuestionResponseEdge!]!
    }

    type QuizQuestionResponseEdge {
        cursor: String!
        node: QuizQuestionResponse!
    }

    extend type Query {
        quizQuestionResponse(input: FindByIdInput!): QuizQuestionResponse
        quizQuestionResponses(input: QuizQuestionResponseSearchInput!): QuizQuestionResponseSearchResult!
    }
`;

export const resolvers: {
    QuizQuestionResponseSortBy: typeof QuizQuestionResponseSortBy;
    Query: EndpointsQuizQuestionResponse["Query"];
} = {
    QuizQuestionResponseSortBy,
    ...QuizQuestionResponseEndpoints,
};
