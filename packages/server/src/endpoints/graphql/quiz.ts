import { QuizSortBy } from "@local/shared";
import { EndpointsQuiz, QuizEndpoints } from "../logic/quiz";

export const typeDef = `#graphql
    enum QuizSortBy {
        AttemptsAsc
        AttemptsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        QuestionsAsc
        QuestionsDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
    }

    input QuizCreateInput {
        id: ID!
        isPrivate: Boolean!
        maxAttempts: Int
        randomizeQuestionOrder: Boolean
        revealCorrectAnswers: Boolean
        timeLimit: Int
        pointsToPass: Int
        projectConnect: ID
        routineConnect: ID
        translationsCreate: [QuizTranslationCreateInput!]
        quizQuestionsCreate: [QuizQuestionCreateInput!]
    }
    input QuizUpdateInput {
        id: ID!
        isPrivate: Boolean
        maxAttempts: Int
        randomizeQuestionOrder: Boolean
        revealCorrectAnswers: Boolean
        timeLimit: Int
        pointsToPass: Int
        routineConnect: ID
        routineDisconnect: Boolean
        projectConnect: ID
        projectDisconnect: Boolean
        translationsCreate: [QuizTranslationCreateInput!]
        translationsUpdate: [QuizTranslationUpdateInput!]
        translationsDelete: [ID!]
        quizQuestionsCreate: [QuizQuestionCreateInput!]
        quizQuestionsUpdate: [QuizQuestionUpdateInput!]
        quizQuestionsDelete: [ID!]
    }
    type Quiz {
        id: ID!
        created_at: Date!
        updated_at: Date!
        isPrivate: Boolean!
        randomizeQuestionOrder: Boolean!
        score: Int!
        bookmarks: Int!
        attempts: [QuizAttempt!]!
        attemptsCount: Int!
        createdBy: User
        project: Project
        quizQuestions: [QuizQuestion!]!
        quizQuestionsCount: Int!
        routine: Routine
        bookmarkedBy: [User!]!
        stats: [StatsQuiz!]!
        translations: [QuizTranslation!]!
        you: QuizYou!
    }

    type QuizYou {
        canDelete: Boolean!
        canBookmark: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        canReact: Boolean!
        hasCompleted: Boolean!
        isBookmarked: Boolean!
        reaction: String
    }

    input QuizTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input QuizTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    type QuizTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input QuizSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        translationLanguages: [String!]
        maxBookmarks: Int
        maxScore: Int
        minBookmarks: Int
        minScore: Int
        routineId: ID
        projectId: ID
        userId: ID
        searchString: String
        sortBy: QuizSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type QuizSearchResult {
        pageInfo: PageInfo!
        edges: [QuizEdge!]!
    }

    type QuizEdge {
        cursor: String!
        node: Quiz!
    }

    extend type Query {
        quiz(input: FindByIdInput!): Quiz
        quizzes(input: QuizSearchInput!): QuizSearchResult!
    }

    extend type Mutation {
        quizCreate(input: QuizCreateInput!): Quiz!
        quizUpdate(input: QuizUpdateInput!): Quiz!
    }
`;

export const resolvers: {
    QuizSortBy: typeof QuizSortBy;
    Query: EndpointsQuiz["Query"];
    Mutation: EndpointsQuiz["Mutation"];
} = {
    QuizSortBy,
    ...QuizEndpoints,
};
