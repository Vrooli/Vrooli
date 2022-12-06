import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Question, QuestionSearchInput, QuestionCreateInput, QuestionUpdateInput, QuestionSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum QuestionSortBy {
        AnswersAsc
        AnswersDesc
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input QuestionCreateInput {
        anonymous: Boolean
        tag: String!
        translationsCreate: [QuestionTranslationCreateInput!]
    }
    input QuestionUpdateInput {
        anonymous: Boolean
        tag: String!
        translationsDelete: [ID!]
        translationsCreate: [QuestionTranslationCreateInput!]
        translationsUpdate: [QuestionTranslationUpdateInput!]
    }

    # User's hidden topics
    input QuestionHiddenCreateInput {
        id: ID!
        isBlur: Boolean
        tagCreate: QuestionCreateInput
        tagConnect: ID
    }

    input QuestionHiddenUpdateInput {
        id: ID!
        isBlur: Boolean
    }

    type Question {
        id: ID!
        tag: String!
        created_at: Date!
        updated_at: Date!
        stars: Int!
        isStarred: Boolean!
        isOwn: Boolean!
        starredBy: [User!]!
        translations: [QuestionTranslation!]!
    }

    input QuestionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
    }
    input QuestionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type QuestionTranslation {
        id: ID!
        language: String!
        description: String
    }

    # Wraps tag with hidden/blurred option
    type QuestionHidden {
        id: ID!
        isBlur: Boolean!
        tag: Question!
    }

    input QuestionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        hidden: Boolean
        ids: [ID!]
        languages: [String!]
        minStars: Int
        myQuestions: Boolean
        searchString: String
        sortBy: QuestionSortBy
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type QuestionSearchResult {
        pageInfo: PageInfo!
        edges: [QuestionEdge!]!
    }

    type QuestionEdge {
        cursor: String!
        node: Question!
    }

    extend type Query {
        question(input: FindByIdInput!): Question
        questions(input: QuestionSearchInput!): QuestionSearchResult!
    }

    extend type Mutation {
        questionCreate(input: QuestionCreateInput!): Question!
        questionUpdate(input: QuestionUpdateInput!): Question!
    }
`

const objectType = 'Question';
export const resolvers: {
    QuestionSortBy: typeof QuestionSortBy;
    Query: {
        question: GQLEndpoint<FindByIdInput, FindOneResult<Question>>;
        questions: GQLEndpoint<QuestionSearchInput, FindManyResult<Question>>;
    },
    Mutation: {
        questionCreate: GQLEndpoint<QuestionCreateInput, CreateOneResult<Question>>;
        questionUpdate: GQLEndpoint<QuestionUpdateInput, UpdateOneResult<Question>>;
    }
} = {
    QuestionSortBy,
    Query: {
        question: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        questions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        questionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        questionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}