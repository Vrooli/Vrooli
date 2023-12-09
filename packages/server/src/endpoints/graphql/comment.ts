import { CommentFor, CommentSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { CommentEndpoints, EndpointsComment } from "../logic/comment";
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
        language: String!
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

export const resolvers: {
    CommentFor: typeof CommentFor;
    CommentSortBy: typeof CommentSortBy;
    CommentedOn: UnionResolver;
    Query: EndpointsComment["Query"];
    Mutation: EndpointsComment["Mutation"];
} = {
    CommentFor,
    CommentSortBy,
    CommentedOn: { __resolveType(obj) { return resolveUnion(obj); } },
    ...CommentEndpoints,
};
