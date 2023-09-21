import { StandardSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsStandard, StandardEndpoints } from "../logic";

export const typeDef = gql`
    enum StandardSortBy {
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

    input StandardCreateInput {
        id: ID!
        isInternal: Boolean
        isPrivate: Boolean!
        permissions: String
        parentConnect: ID
        ownedByUserConnect: ID
        ownedByOrganizationConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [StandardVersionCreateInput!]
    }
    input StandardUpdateInput {
        id: ID!
        isInternal: Boolean
        isPrivate: Boolean
        permissions: String
        ownedByUserConnect: ID
        ownedByOrganizationConnect: ID
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [StandardVersionCreateInput!]
        versionsUpdate: [StandardVersionUpdateInput!]
        versionsDelete: [ID!]
    }
    type Standard {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompleteVersion: Boolean!
        isDeleted: Boolean!
        isInternal: Boolean!
        isPrivate: Boolean!
        permissions: String!
        translatedName: String!
        score: Int!
        bookmarks: Int!
        views: Int!
        createdBy: User
        forks: [Standard!]!
        forksCount: Int!
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        owner: Owner
        parent: StandardVersion
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsStandard!]!
        tags: [Tag!]!
        transfers: [Transfer!]!
        transfersCount: Int!
        versions: [StandardVersion!]!
        versionsCount: Int
        you: StandardYou!
    }

    type StandardYou {
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

    input StandardSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        hasCompleteVersion: Boolean
        isInternal: Boolean
        issuesId: ID
        labelsIds: [ID!]
        maxScore: Int
        maxBookmarks: Int
        maxViews: Int
        minScore: Int
        minBookmarks: Int
        minViews: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
        pullRequestsId: ID
        searchString: String
        sortBy: StandardSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }
    type StandardSearchResult {
        pageInfo: PageInfo!
        edges: [StandardEdge!]!
    }
    type StandardEdge {
        cursor: String!
        node: Standard!
    }

    extend type Query {
        standard(input: FindByIdInput!): Standard
        standards(input: StandardSearchInput!): StandardSearchResult!
    }

    extend type Mutation {
        standardCreate(input: StandardCreateInput!): Standard!
        standardUpdate(input: StandardUpdateInput!): Standard!
    }
`;

export const resolvers: {
    StandardSortBy: typeof StandardSortBy;
    Query: EndpointsStandard["Query"];
    Mutation: EndpointsStandard["Mutation"];
} = {
    StandardSortBy,
    ...StandardEndpoints,
};
