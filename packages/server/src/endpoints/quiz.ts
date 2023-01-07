import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, QuizSortBy, Quiz, QuizSearchInput, QuizCreateInput, QuizUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
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
        StarsAsc
        StarsDesc
    }

    input QuizCreateInput {
        id: ID!
        maxAttempts: Int
        randomizeQuestionORder: Boolean
        revealCorrectAnswers: Boolean
        timeLimit: Int
        pointsToPass: Int
        routineConnect: ID
        projectConnect: ID
        translationsCreate: [QuizTranslationCreateInput!]
        quizQuestionsCreate: [QuizQuestionCreateInput!]
    }
    input QuizUpdateInput {
        id: ID!
        maxAttempts: Int
        randomizeQuestionORder: Boolean
        revealCorrectAnswers: Boolean
        timeLimit: Int
        pointsToPass: Int
        routineConnect: ID
        routineDisconnect: ID
        projectConnect: ID
        projectDisconnect: ID
        translationsCreate: [QuizTranslationCreateInput!]
        translationsUpdate: [QuizTranslationUpdateInput!]
        translationsDelete: [ID!]
        quizQuestionsCreate: [QuizQuestionCreateInput!]
        quizQuestionsUpdate: [QuizQuestionUpdateInput!]
        quizQuestionsDelete: [ID!]
    }
    type Quiz {
        type: GqlModelType!
        id: ID!
        created_at: Date!
        updated_at: Date!
        isCompleted: Boolean!
        score: Int!
        stars: Int!
        views: Int!
        attempts: [QuizAttempt!]!
        createdBy: User
        project: Project
        quizQuestions: [QuizQuestion!]!
        routine: Routine
        starredBy: [User!]!
        stats: [StatsQuiz!]!
        translations: [QuizTranslation!]!
        you: QuizYou!
    }

    type QuizYou {
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canView: Boolean!
        canVote: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
    }

    input QuizTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input QuizTranslationUpdateInput {
        id: ID!
        language: String
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
        languages: [String!]
        minScore: Int
        minStars: Int
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
`

const objectType = 'Quiz';
export const resolvers: {
    QuizSortBy: typeof QuizSortBy;
    Query: {
        quiz: GQLEndpoint<FindByIdInput, FindOneResult<Quiz>>;
        quizzes: GQLEndpoint<QuizSearchInput, FindManyResult<Quiz>>;
    },
    Mutation: {
        quizCreate: GQLEndpoint<QuizCreateInput, CreateOneResult<Quiz>>;
        quizUpdate: GQLEndpoint<QuizUpdateInput, UpdateOneResult<Quiz>>;
    }
} = {
    QuizSortBy,
    Query: {
        quiz: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        quizzes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        quizCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        quizUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}