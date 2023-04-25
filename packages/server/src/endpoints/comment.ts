import { Comment, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentUpdateInput, FindByIdInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
import { CommentModel } from "../models";
import { CreateOneResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from "../types";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
    enum CommentFor {
        ApiVersion
        Issue
        NoteVersion
        Post
        ProjectVersion
        PullRequest
        Question
        QuestionAnswer
        RoutineVersion
        SmartContractVersion
        StandardVersion
    }   

    enum CommentSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
    }

    input CommentCreateInput {
        id: ID!
        createdFor: CommentFor!
        forConnect: ID!
        parentConnect: ID
        translationsCreate: [CommentTranslationCreateInput!]
    }
    input CommentUpdateInput {
        id: ID!
        translationsDelete: [ID!]
        translationsCreate: [CommentTranslationCreateInput!]
        translationsUpdate: [CommentTranslationUpdateInput!]
    }

    union CommentedOn = ApiVersion | Issue | NoteVersion | Post | ProjectVersion | PullRequest | Question | QuestionAnswer | RoutineVersion | SmartContractVersion | StandardVersion

    type Comment {
        id: ID!
        created_at: Date!
        updated_at: Date!
        commentedOn: CommentedOn!
        owner: Owner
        reports: [Report!]!
        reportsCount: Int!
        score: Int!
        bookmarks: Int!
        bookmarkedBy: [User!]
        translations: [CommentTranslation!]!
        translationsCount: Int!
        you: CommentYou!
    }

    type CommentYou {
        canDelete: Boolean!
        canUpdate: Boolean!
        canBookmark: Boolean!
        canReply: Boolean!
        canReport: Boolean!
        canReact: Boolean!
        isBookmarked: Boolean!
        reaction: String
    }

    input CommentTranslationCreateInput {
        id: ID!
        language: String!
        text: String!
    }
    input CommentTranslationUpdateInput {
        id: ID!
        language: String
        text: String
    }
    type CommentTranslation {
        id: ID!
        language: String!
        text: String!
    }

    input CommentSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        minScore: Int
        minBookmarks: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        searchString: String
        sortBy: CommentSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        apiVersionId: ID
        issueId: ID
        noteVersionId: ID
        postId: ID
        projectVersionId: ID
        pullRequestId: ID
        questionId: ID
        questionAnswerId: ID
        routineVersionId: ID
        smartContractVersionId: ID
        standardVersionId: ID
        translationLanguages: [String!]
    }

    type CommentThread {
        childThreads: [CommentThread!]!
        comment: Comment!
        endCursor: String
        totalInThread: Int
    }

    type CommentSearchResult {
        endCursor: String
        threads: [CommentThread!]
        totalThreads: Int
    }

    extend type Query {
        comment(input: FindByIdInput!): Comment
        comments(input: CommentSearchInput!): CommentSearchResult!
    }

    extend type Mutation {
        commentCreate(input: CommentCreateInput!): Comment!
        commentUpdate(input: CommentUpdateInput!): Comment!
    }
`;

const objectType = "Comment";
export const resolvers: {
    CommentFor: typeof CommentFor;
    CommentSortBy: typeof CommentSortBy;
    CommentedOn: UnionResolver;
    Query: {
        comment: GQLEndpoint<FindByIdInput, FindOneResult<Comment>>;
        comments: GQLEndpoint<CommentSearchInput, CommentSearchResult>;
    },
    Mutation: {
        commentCreate: GQLEndpoint<CommentCreateInput, CreateOneResult<Comment>>;
        commentUpdate: GQLEndpoint<CommentUpdateInput, UpdateOneResult<Comment>>;
    }
} = {
    CommentFor,
    CommentSortBy,
    CommentedOn: { __resolveType(obj) { return resolveUnion(obj); } },
    Query: {
        comment: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        comments: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return CommentModel.query.searchNested(prisma, req, input, info);
        },
    },
    Mutation: {
        commentCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        commentUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
