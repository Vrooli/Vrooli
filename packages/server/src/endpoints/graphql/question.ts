import { QuestionForType, QuestionSortBy } from "@local/shared";
import { UnionResolver } from "../../types";
import { EndpointsQuestion, QuestionEndpoints } from "../logic/question";
import { resolveUnion } from "./resolvers";

export const typeDef = `#graphql
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
        BookmarksAsc
        BookmarksDesc
    }

    enum QuestionForType {
        Api
        Code
        Note
        Project
        Routine
        Standard
        Team
    }

    union QuestionFor = Api | Code | Note | Project | Routine | Standard | Team

    input QuestionCreateInput {
        id: ID!
        isPrivate: Boolean!
        referencing: String
        forObjectType: QuestionForType
        forObjectConnect: ID
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [QuestionTranslationCreateInput!]
    }
    input QuestionUpdateInput {
        id: ID!
        isPrivate: Boolean
        acceptedAnswerConnect: ID
        acceptedAnswerDisconnect: Boolean
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
        isPrivate: Boolean!
        bookmarks: Int!
        score: Int!
        forObject: QuestionFor
        reports: [Report!]!
        reportsCount: Int!
        translations: [QuestionTranslation!]!
        translationsCount: Int!
        answers: [QuestionAnswer!]!
        answersCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        bookmarkedBy: [User!]!
        tags: [Tag!]!
        you: QuestionYou!
    }

    type QuestionYou {
        canDelete: Boolean!
        canBookmark: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        canReact: Boolean!
        isBookmarked: Boolean!
        reaction: String
    }

    input QuestionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input QuestionTranslationUpdateInput {
        id: ID!
        language: String!
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
        codeId: ID
        noteId: ID
        projectId: ID
        routineId: ID
        standardId: ID
        teamId: ID
        ids: [ID!]
        translationLanguages: [String!]
        maxScore: Int
        maxBookmarks: Int
        minScore: Int
        minBookmarks: Int
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
`;

export const resolvers: {
    QuestionSortBy: typeof QuestionSortBy;
    QuestionForType: typeof QuestionForType;
    QuestionFor: UnionResolver;
    Query: EndpointsQuestion["Query"];
    Mutation: EndpointsQuestion["Mutation"];
} = {
    QuestionSortBy,
    QuestionForType,
    QuestionFor: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...QuestionEndpoints,
};
