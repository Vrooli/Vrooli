import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from '../types';
import { FindByIdInput, IssueSortBy, Issue, IssueSearchInput, IssueCreateInput, IssueUpdateInput, IssueStatus, IssueFor, IssueCloseInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { resolveUnion } from './resolvers';
import { CustomError } from '../events';

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
        StarsAsc
        StarsDesc
    }

    enum IssueStatus {
        Open
        ClosedResolved
        CloseUnresolved
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

    union IssueTo = Api | Organization | Note | Project | Routine | SmartContract | Standard

    input IssueCreateInput {
        id: ID!
        issueFor: IssueFor!
        forConnect: ID!
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
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
        referencedVersionConnect: ID
        comments: [Comment!]!
        commentsCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        reports: [Report!]!
        reportsCount: Int!
        translations: [IssueTranslation!]!
        translationsCount: Int!
        score: Int!
        stars: Int!
        views: Int!
        starredBy: [Star!]
        you: IssueYou!
    }

    type IssueYou {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
    }

    input IssueTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input IssueTranslationUpdateInput {
        id: ID!
        language: String
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
        minStars: Int
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
`

const objectType = 'Issue';
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
    IssueTo: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        issue: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        issues: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        issueCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        issueUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        issueClose: async (_, { input }, { prisma, req }, info) => {
            throw new CustomError('0000', 'NotImplemented', ['en'])
        },
    }
}