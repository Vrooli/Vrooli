import { ProjectSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsProject, ProjectEndpoints } from "../logic/project";

export const typeDef = gql`
    enum ProjectSortBy {
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

    input ProjectCreateInput {
        id: ID!
        handle: String
        isPrivate: Boolean!
        permissions: String
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        parentConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [ProjectVersionCreateInput!]
    }
    input ProjectUpdateInput {
        id: ID!
        handle: String
        isPrivate: Boolean
        permissions: String
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        versionsCreate: [ProjectVersionCreateInput!]
        versionsUpdate: [ProjectVersionUpdateInput!]
        versionsDelete: [ID!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    type Project {
        id: ID!
        created_at: Date!
        updated_at: Date!
        handle: String
        hasCompleteVersion: Boolean!
        permissions: String!
        isPrivate: Boolean!
        translatedName: String!
        score: Int!
        views: Int!
        bookmarks: Int!
        createdBy: User
        owner: Owner
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        parent: ProjectVersion
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        quizzes: [Quiz!]!
        quizzesCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsProject!]!
        tags: [Tag!]!
        transfers: [Transfer!]!
        transfersCount: Int!
        versions: [ProjectVersion!]!
        versionsCount: Int!
        you: ProjectYou!
    }

    type ProjectYou {
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

    input ProjectSearchInput {
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
        sortBy: ProjectSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ProjectSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectEdge!]!
    }

    type ProjectEdge {
        cursor: String!
        node: Project!
    }

    extend type Query {
        project(input: FindByIdOrHandleInput!): Project
        projects(input: ProjectSearchInput!): ProjectSearchResult!
    }

    extend type Mutation {
        projectCreate(input: ProjectCreateInput!): Project!
        projectUpdate(input: ProjectUpdateInput!): Project!
    }
`;

export const resolvers: {
    ProjectSortBy: typeof ProjectSortBy;
    Query: EndpointsProject["Query"];
    Mutation: EndpointsProject["Mutation"];
} = {
    ProjectSortBy,
    ...ProjectEndpoints,
};
