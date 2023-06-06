import { SmartContractSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsSmartContract, SmartContractEndpoints } from "../logic";

export const typeDef = gql`
    enum SmartContractSortBy {
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

    input SmartContractCreateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String
        parentConnect: ID
        ownedByUserConnect: ID
        ownedByOrganizationConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [SmartContractVersionCreateInput!]
    }
    input SmartContractUpdateInput {
        id: ID!
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
        versionsCreate: [SmartContractVersionCreateInput!]
        versionsUpdate: [SmartContractVersionUpdateInput!]
        versionsDelete: [ID!]
    }
    type SmartContract {
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
        parent: SmartContractVersion
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsSmartContract!]!
        tags: [Tag!]!
        transfers: [Transfer!]!
        transfersCount: Int!
        versions: [SmartContractVersion!]!
        versionsCount: Int
        you: SmartContractYou!
    }

    type SmartContractYou {
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

    input SmartContractSearchInput {
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
        sortBy: SmartContractSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type SmartContractSearchResult {
        pageInfo: PageInfo!
        edges: [SmartContractEdge!]!
    }

    type SmartContractEdge {
        cursor: String!
        node: SmartContract!
    }

    extend type Query {
        smartContract(input: FindByIdInput!): SmartContract
        smartContracts(input: SmartContractSearchInput!): SmartContractSearchResult!
    }

    extend type Mutation {
        smartContractCreate(input: SmartContractCreateInput!): SmartContract!
        smartContractUpdate(input: SmartContractUpdateInput!): Api!
    }
`;

export const resolvers: {
    SmartContractSortBy: typeof SmartContractSortBy;
    Query: EndpointsSmartContract["Query"];
    Mutation: EndpointsSmartContract["Mutation"];
} = {
    SmartContractSortBy,
    ...SmartContractEndpoints,
};
