import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from '../types';
import { FindByIdInput, Question, QuestionSearchInput, QuestionCreateInput, QuestionUpdateInput, QuestionSortBy, QuestionForType } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { resolveUnion } from './resolvers';

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
        ScoreAsc
        ScoreDesc
        StarsAsc
        StarsDesc
    }

    enum QuestionForType {
        Api
        Note
        Organization
        Project
        Routine
        SmartContract
        Standard
    }

    union QuestionFor = Api | Note | Organization | Project | Routine | SmartContract | Standard

    input QuestionCreateInput {
        id: ID!
        referencing: String
        forType: QuestionForType!
        forConnect: ID!
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [QuestionTranslationCreateInput!]
    }
    input QuestionUpdateInput {
        id: ID!
        acceptedAnswerConnect: ID
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [QuestionTranslationCreateInput!]
        translationsUpdate: [QuestionTranslationUpdateInput!]
    }
    type Question {
        id: ID!
        created_at: Date!
        updated_at: Date!
        createdBy: User
        hasAcceptedAnswer: Boolean!
        score: Int!
        stars: Int!
        forObject: QuestionFor!
        reports: [Report!]!
        reportsCount: Int!
        translations: [QuestionTranslation!]!
        translationsCount: Int!
        answers: [QuestionAnswer!]!
        answersCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        starredBy: [User!]!
        tags: [Tag!]!
        you: QuestionYou!
    }

    type QuestionYou {
        canDelete: Boolean!
        canStar: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        canVote: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
    }

    input QuestionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input QuestionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type QuestionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input QuestionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        hasAcceptedAnswer: Boolean
        createdById: ID
        apiId: ID
        noteId: ID
        organizationId: ID
        projectId: ID
        routineId: ID
        smartContractId: ID
        standardId: ID
        ids: [ID!]
        languages: [String!]
        maxScore: Int
        maxStars: Int
        minScore: Int
        minStars: Int
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
    QuestionForType: typeof QuestionForType;
    QuestionFor: UnionResolver;
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
    QuestionForType,
    QuestionFor: { __resolveType(obj: any) { return resolveUnion(obj) } },
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