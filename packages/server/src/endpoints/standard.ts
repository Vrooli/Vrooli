import { FindByIdInput, Standard, StandardSortBy, StandardVersion, StandardVersionCreateInput, StandardVersionSearchInput, StandardVersionUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

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
        isPrivate: Boolean
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

const objectType = "Standard";
export const resolvers: {
    StandardSortBy: typeof StandardSortBy;
    Query: {
        standard: GQLEndpoint<FindByIdInput, FindOneResult<Standard>>;
        standards: GQLEndpoint<StandardVersionSearchInput, FindManyResult<StandardVersion>>;
    },
    Mutation: {
        standardCreate: GQLEndpoint<StandardVersionCreateInput, CreateOneResult<StandardVersion>>;
        standardUpdate: GQLEndpoint<StandardVersionUpdateInput, UpdateOneResult<StandardVersion>>;
    }
} = {
    StandardSortBy,
    Query: {
        standard: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        standards: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        standardCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        standardUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
