import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, QuizSortBy, QuizAttemptStatus, QuizAttempt, QuizAttemptSearchInput, QuizAttemptCreateInput, QuizAttemptUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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
    }

    type QuizAttemptPermission {
        canDelete: Boolean!
        canEdit: Boolean!
    }

    input QuizAttemptSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: QuizAttemptStatus
        languageIn: [String!]
        userId: ID
        quizId: ID
        searchString: String
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
`

const objectType = 'QuizAttempt';
export const resolvers: {
    QuizSortBy: typeof QuizSortBy;
    QuizAttemptStatus: typeof QuizAttemptStatus;
    Query: {
        quizAttempt: GQLEndpoint<FindByIdInput, FindOneResult<QuizAttempt>>;
        quizAttempts: GQLEndpoint<QuizAttemptSearchInput, FindManyResult<QuizAttempt>>;
    },
    Mutation: {
        quizAttemptCreate: GQLEndpoint<QuizAttemptCreateInput, CreateOneResult<QuizAttempt>>;
        quizAttemptUpdate: GQLEndpoint<QuizAttemptUpdateInput, UpdateOneResult<QuizAttempt>>;
    }
} = {
    QuizSortBy,
    QuizAttemptStatus,
    Query: {
        quizAttempt: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        quizAttempts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        quizAttemptCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        quizAttemptUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}