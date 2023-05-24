import { FindByIdInput, Issue, IssueCloseInput, IssueCreateInput, IssueFor, IssueSearchInput, IssueSortBy, IssueStatus, IssueUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { CustomError } from "../events";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from "../types";
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
        Open
        ClosedResolved
        ClosedUnresolved
        Rejected
    }

    enum IssueFor {
        Api
        Organization
        Note
        Project
        Routine
        SmartContract
        Standard
    } 

    union IssueTo = Api | Note | Organization | Project | Routine | SmartContract | Standard

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
        organizationId: ID
        noteId: ID
        projectId: ID
        routineId: ID
        smartContractId: ID
        standardId: ID
        closedById: ID
        createdById: ID
        referencedVersionId: ID
        searchString: String
        sortBy: IssueSortBy
        take: Int
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

const objectType = "Issue";
export const resolvers: {
    IssueSortBy: typeof IssueSortBy;
    IssueStatus: typeof IssueStatus;
    IssueFor: typeof IssueFor;
    IssueTo: UnionResolver;
    Query: {
        issue: GQLEndpoint<FindByIdInput, FindOneResult<Issue>>;
        issues: GQLEndpoint<IssueSearchInput, FindManyResult<Issue>>;
    },
    Mutation: {
        issueCreate: GQLEndpoint<IssueCreateInput, CreateOneResult<Issue>>;
        issueUpdate: GQLEndpoint<IssueUpdateInput, UpdateOneResult<Issue>>;
        issueClose: GQLEndpoint<IssueCloseInput, UpdateOneResult<Issue>>;
    }
} = {
    IssueSortBy,
    IssueStatus,
    IssueFor,
    IssueTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    Query: {
        issue: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        issues: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        issueCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        issueUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        issueClose: async (_, { input }, { prisma, req }, info) => {
            throw new CustomError("0000", "NotImplemented", ["en"]);
            // TODO make sure to set hasBeenClosedOrRejected to true
        },
    },
};
