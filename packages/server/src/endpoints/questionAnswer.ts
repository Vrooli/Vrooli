import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, QuestionAnswer, QuestionAnswerSearchInput, QuestionAnswerCreateInput, QuestionAnswerUpdateInput, QuestionAnswerSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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
        StarsAsc
        StarsDesc
    }

    input QuestionAnswerCreateInput {
        anonymous: Boolean
        tag: String!
        translationsCreate: [QuestionAnswerTranslationCreateInput!]
    }
    input QuestionAnswerUpdateInput {
        anonymous: Boolean
        tag: String!
        translationsDelete: [ID!]
        translationsCreate: [QuestionAnswerTranslationCreateInput!]
        translationsUpdate: [QuestionAnswerTranslationUpdateInput!]
    }

    # User's hidden topics
    input QuestionAnswerHiddenCreateInput {
        id: ID!
        isBlur: Boolean
        tagCreate: QuestionAnswerCreateInput
        tagConnect: ID
    }

    input QuestionAnswerHiddenUpdateInput {
        id: ID!
        isBlur: Boolean
    }

    type QuestionAnswer {
        id: ID!
        tag: String!
        created_at: Date!
        updated_at: Date!
        stars: Int!
        isStarred: Boolean!
        isOwn: Boolean!
        starredBy: [User!]!
        translations: [QuestionAnswerTranslation!]!
    }

    input QuestionAnswerTranslationCreateInput {
        id: ID!
        language: String!
        description: String
    }
    input QuestionAnswerTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type QuestionAnswerTranslation {
        id: ID!
        language: String!
        description: String
    }

    # Wraps tag with hidden/blurred option
    type QuestionAnswerHidden {
        id: ID!
        isBlur: Boolean!
        tag: QuestionAnswer!
    }

    input QuestionAnswerSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        hidden: Boolean
        ids: [ID!]
        languages: [String!]
        minStars: Int
        myQuestionAnswers: Boolean
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
    }
`

const objectType = 'QuestionAnswer';
export const resolvers: {
    QuestionAnswerSortBy: typeof QuestionAnswerSortBy;
    Query: {
        questionAnswer: GQLEndpoint<FindByIdInput, FindOneResult<QuestionAnswer>>;
        questionAnswers: GQLEndpoint<QuestionAnswerSearchInput, FindManyResult<QuestionAnswer>>;
    },
    Mutation: {
        questionAnswerCreate: GQLEndpoint<QuestionAnswerCreateInput, CreateOneResult<QuestionAnswer>>;
        questionAnswerUpdate: GQLEndpoint<QuestionAnswerUpdateInput, UpdateOneResult<QuestionAnswer>>;
    }
} = {
    QuestionAnswerSortBy,
    Query: {
        questionAnswer: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        questionAnswers: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        questionAnswerCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        questionAnswerUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}