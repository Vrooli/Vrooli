import { CodeSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { CodeEndpoints, EndpointsCode } from "../logic/code";

export const typeDef = gql`
    enum CodeSortBy {
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IssuesAsc
        IssuesDesc
        PullRequestsAsc
        PullRequestsDesc
        QuestionsAsc
        QuestionsDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    input CodeCreateInput {
        id: ID!
        isPrivate: Boolean!
        permissions: String
        parentConnect: ID
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [CodeVersionCreateInput!]
    }
    input CodeUpdateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [CodeVersionCreateInput!]
        versionsUpdate: [CodeVersionUpdateInput!]
        versionsDelete: [ID!]
    }
    type Code {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompleteVersion: Boolean!
        isDeleted: Boolean!
        isPrivate: Boolean!
        permissions: String!
        translatedName: String!
        score: Int!
        bookmarks: Int!
        views: Int!
        createdBy: User
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        owner: Owner
        parent: CodeVersion
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsCode!]!
        tags: [Tag!]!
        transfers: [Transfer!]!
        transfersCount: Int!
        versions: [CodeVersion!]!
        versionsCount: Int
        you: CodeYou!
    }

    type CodeYou {
        canDelete: Boolean!
        canBookmark: Boolean!
        canTransfer: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        canReact: Boolean!
        isBookmarked: Boolean!
        reaction: String
        isViewed: Boolean!
    }

    input CodeSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        hasCompleteVersion: Boolean
        issuesId: ID
        labelsIds: [ID!]
        maxScore: Int
        maxBookmarks: Int
        maxViews: Int
        minScore: Int
        minBookmarks: Int
        minViews: Int
        ownedByTeamId: ID
        ownedByUserId: ID
        parentId: ID
        pullRequestsId: ID
        searchString: String
        sortBy: CodeSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type CodeSearchResult {
        pageInfo: PageInfo!
        edges: [CodeEdge!]!
    }

    type CodeEdge {
        cursor: String!
        node: Code!
    }

    extend type Query {
        code(input: FindByIdInput!): Code
        codes(input: CodeSearchInput!): CodeSearchResult!
    }

    extend type Mutation {
        codeCreate(input: CodeCreateInput!): Code!
        codeUpdate(input: CodeUpdateInput!): Api!
    }
`;

export const resolvers: {
    CodeSortBy: typeof CodeSortBy;
    Query: EndpointsCode["Query"];
    Mutation: EndpointsCode["Mutation"];
} = {
    CodeSortBy,
    ...CodeEndpoints,
};
