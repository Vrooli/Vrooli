import { gql } from 'apollo-server-express';
import { FindManyResult, FindOneResult, GQLEndpoint } from '../types';
import { FindByIdInput, QuizSortBy, QuizQuestionResponse, QuizQuestionResponseSearchInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { readManyHelper, readOneHelper } from '../actions';

export const typeDef = gql`
    enum QuizQuestionResponseSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input QuizQuestionResponseCreateInput {
        id: ID!
        response: String
        quizAttemptConnect: ID!
        quizQuestionConnect: ID!
        translationsCreate: [QuizQuestionResponseTranslationCreateInput!]
    }
    input QuizQuestionResponseUpdateInput {
        id: ID!
        translationsCreate: [QuizQuestionResponseTranslationCreateInput!]
        translationsUpdate: [QuizQuestionResponseTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type QuizQuestionResponse {
        id: ID!
        created_at: Date!
        updated_at: Date!
        response: String
        quizAttempt: QuizAttempt!
        quizQuestion: QuizQuestion!
        permissionsQuizQuestionResponse: QuizQuestionResponsePermission!
    }

    type QuizQuestionResponsePermission {
        canDelete: Boolean!
        canEdit: Boolean!
    }

    input QuizQuestionResponseTranslationCreateInput {
        id: ID!
        language: String!
        response: String!
    }
    input QuizQuestionResponseTranslationUpdateInput {
        id: ID!
        language: String
        response: String
    }
    type QuizQuestionResponseTranslation {
        id: ID!
        language: String!
        response: String!
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
`

const objectType = 'QuizQuestionResponse';
export const resolvers: {
    QuizSortBy: typeof QuizSortBy;
    Query: {
        quizQuestionResponse: GQLEndpoint<FindByIdInput, FindOneResult<QuizQuestionResponse>>;
        quizQuestionResponses: GQLEndpoint<QuizQuestionResponseSearchInput, FindManyResult<QuizQuestionResponse>>;
    },
} = {
    QuizSortBy,
    Query: {
        quizQuestionResponse: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        quizQuestionResponses: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
}