import { IssueFor, IssueSortBy, IssueStatus } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { EndpointsIssue, IssueEndpoints } from "../logic/issue";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
    enum IssueSortBy {
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
    }

    enum IssueStatus {
        Draft
        Open
        ClosedResolved
        ClosedUnresolved
        Rejected
    }

    enum IssueFor {
        Api
        Code
        Note
        Project
        Routine
        Standard
        Team
    } 

    union IssueTo = Api | Code | Note | Project | Routine | Standard | Team

    input IssueCreateInput {
        id: ID!
        issueFor: IssueFor!
        forConnect: ID!
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        referencedVersionIdConnect: ID
        translationsCreate: [IssueTranslationCreateInput!]
    }
    input IssueUpdateInput {
        id: ID!
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        translationsCreate: [IssueTranslationCreateInput!]
        translationsUpdate: [IssueTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type Issue {
        id: ID!
        created_at: Date!
        updated_at: Date!
        closedAt: Date
        closedBy: User
        createdBy: User
        status: IssueStatus!
        to: IssueTo!
        referencedVersionId: String
        comments: [Comment!]!
        commentsCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        reports: [Report!]!
        reportsCount: Int!
        translations: [IssueTranslation!]!
        translationsCount: Int!
        score: Int!
        bookmarks: Int!
        views: Int!
        bookmarkedBy: [Bookmark!]
        you: IssueYou!
    }

    type IssueYou {
        canComment: Boolean!
        canDelete: Boolean!
        canUpdate: Boolean!
        canBookmark: Boolean!
        canReport: Boolean!
        canRead: Boolean!
        canReact: Boolean!
        isBookmarked: Boolean!
        reaction: String
    }

    input IssueTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input IssueTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    type IssueTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input IssueSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: IssueStatus
        minScore: Int
        minBookmarks: Int
        minViews: Int
        apiId: ID
        codeId: ID
        noteId: ID
        projectId: ID
        routineId: ID
        standardId: ID
        closedById: ID
        createdById: ID
        referencedVersionId: ID
        searchString: String
        sortBy: IssueSortBy
        take: Int
        teamId: ID
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type IssueSearchResult {
        pageInfo: PageInfo!
        edges: [IssueEdge!]!
    }

    type IssueEdge {
        cursor: String!
        node: Issue!
    }

    input IssueCloseInput {
        id: ID!
        status: IssueStatus!
    }

    extend type Query {
        issue(input: FindByIdInput!): Issue
        issues(input: IssueSearchInput!): IssueSearchResult!
    }

    extend type Mutation {
        issueCreate(input: IssueCreateInput!): Issue!
        issueUpdate(input: IssueUpdateInput!): Issue!
        issueClose(input: IssueCloseInput!): Issue!
    }
`;

export const resolvers: {
    IssueSortBy: typeof IssueSortBy;
    IssueStatus: typeof IssueStatus;
    IssueFor: typeof IssueFor;
    IssueTo: UnionResolver;
    Query: EndpointsIssue["Query"];
    Mutation: EndpointsIssue["Mutation"];
} = {
    IssueSortBy,
    IssueStatus,
    IssueFor,
    IssueTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...IssueEndpoints,
};
