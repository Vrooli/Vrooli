import { ApiSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { ApiEndpoints, EndpointsApi } from "../logic";

export const typeDef = gql`
    enum ApiSortBy {
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

    input ApiCreateInput {
        id: ID!
        isPrivate: Boolean!
        permissions: String
        ownedByUserConnect: ID
        ownedByOrganizationConnect: ID
        parentConnect: ID
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [ApiVersionCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    input ApiUpdateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String
        ownedByUserConnect: ID
        ownedByOrganizationConnect: ID
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [ApiVersionCreateInput!]
        versionsUpdate: [ApiVersionUpdateInput!]
        versionsDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    type Api {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompleteVersion: Boolean!
        isDeleted: Boolean!
        isPrivate: Boolean!
        permissions: String!
        createdBy: User
        owner: Owner
        parent: ApiVersion
        tags: [Tag!]!
        versions: [ApiVersion!]!
        versionsCount: Int!
        labels: [Label!]!
        bookmarks: Int!
        views: Int!
        score: Int!
        issues: [Issue!]!
        issuesCount: Int!
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        stats: [StatsApi!]!
        questions: [Question!]!
        questionsCount: Int!
        transfers: [Transfer!]!
        transfersCount: Int!
        bookmarkedBy: [User!]!
        you: ApiYou!
    }

    type ApiYou {
        canDelete: Boolean!
        canUpdate: Boolean!
        canBookmark: Boolean!
        canTransfer: Boolean!
        canRead: Boolean!
        canReact: Boolean!
        isBookmarked: Boolean!
        reaction: String
        isViewed: Boolean!
    }

    input ApiSearchInput {
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
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
        pullRequestsId: ID
        searchString: String
        sortBy: ApiSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ApiSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type ApiEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        api(input: FindByIdInput!): Api
        apis(input: ApiSearchInput!): ApiSearchResult!
    }

    extend type Mutation {
        apiCreate(input: ApiCreateInput!): Api!
        apiUpdate(input: ApiUpdateInput!): Api!
    }
`;


export const resolvers: {
    ApiSortBy: typeof ApiSortBy;
    Query: EndpointsApi["Query"];
    Mutation: EndpointsApi["Mutation"];
} = {
    ApiSortBy,
    ...ApiEndpoints,
};
