import { QuizAttemptStatus, QuizSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsQuizAttempt, QuizAttemptEndpoints } from "../logic";

export const typeDef = gql`
    enum QuizAttemptSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        TimeTakenAsc
        TimeTakenDesc
        PointsEarnedAsc
        PointsEarnedDesc
        ContextSwitchesAsc
        ContextSwitchesDesc
    }

    enum QuizAttemptStatus {
        NotStarted
        InProgress
        Passed
        Failed
    }

    input QuizAttemptCreateInput {
        id: ID!
        contextSwitches: Int
        timeTaken: Int
        language: String!
        quizConnect: ID!
        responsesCreate: [QuizQuestionResponseCreateInput!]
    }
    input QuizAttemptUpdateInput {
        id: ID!
        contextSwitches: Int
        timeTaken: Int
        responsesCreate: [QuizQuestionResponseCreateInput!]
        responsesUpdate: [QuizQuestionResponseUpdateInput!]
        responsesDelete: [ID!]
    }
    type QuizAttempt {
        id: ID!
        created_at: Date!
        updated_at: Date!
        pointsEarned: Int!
        status: QuizAttemptStatus!
        contextSwitches: Int!
        timeTaken: Int
        quiz: Quiz!
        responses: [QuizQuestionResponse!]!
        responsesCount: Int!
        user: User!
        you: QuizAttemptYou!
    }

    type QuizAttemptYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input QuizAttemptSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: QuizAttemptStatus
        languageIn: [String!]
        maxPointsEarned: Int
        minPointsEarned: Int
        userId: ID
        quizId: ID
        sortBy: QuizAttemptSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type QuizAttemptSearchResult {
        pageInfo: PageInfo!
        edges: [QuizAttemptEdge!]!
    }

    type QuizAttemptEdge {
        cursor: String!
        node: QuizAttempt!
    }

    extend type Query {
        quizAttempt(input: FindByIdInput!): QuizAttempt
        quizAttempts(input: QuizAttemptSearchInput!): QuizAttemptSearchResult!
    }

    extend type Mutation {
        quizAttemptCreate(input: QuizAttemptCreateInput!): QuizAttempt!
        quizAttemptUpdate(input: QuizAttemptUpdateInput!): QuizAttempt!
    }
`;

export const resolvers: {
    QuizSortBy: typeof QuizSortBy;
    QuizAttemptStatus: typeof QuizAttemptStatus;
    Query: EndpointsQuizAttempt["Query"];
    Mutation: EndpointsQuizAttempt["Mutation"];
} = {
    QuizSortBy,
    QuizAttemptStatus,
    ...QuizAttemptEndpoints,
};
