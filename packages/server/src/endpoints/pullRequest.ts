import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from '../types';
import { FindByIdInput, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, PullRequestSortBy, PullRequestToObjectType, PullRequestStatus, PullRequest, PullRequestSearchInput, PullRequestCreateInput, PullRequestUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { resolveUnion } from './resolvers';

export const typeDef = gql`
    enum PullRequestSortBy {
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum PullRequestToObjectType {
        Api
        Note
        Project
        Routine
        SmartContract
        Standard
    }

    enum PullRequestStatus {
        Open
        Merged
        Rejected
    }

    union PullRequestTo = Api | Note | Project | Routine | SmartContract | Standard

    union PullRequestFrom = ApiVersion | NoteVersion | ProjectVersion | RoutineVersion | SmartContractVersion | StandardVersion

    input PullRequestCreateInput {
        id: ID!
        toObjectType: PullRequestToObjectType!
        toId: ID!
        fromId: ID!
    }
    input PullRequestUpdateInput {
        id: ID!
        status: PullRequestStatus
    }
    type PullRequest {
        id: ID!
        created_at: Date!
        updated_at: Date!
        mergedOrRejectedAt: Date
        status: PullRequestStatus!
        from: PullRequestFrom!
        to: PullRequestTo!
        createdBy: User
        comments: [Comment!]!
        permissionsPullRequest: PullRequestPermission!
    }

    type PullRequestPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canReport: Boolean!
    }

    input PullRequestSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isMergedOrRejected: Boolean
        languages: [String!]
        toId: ID
        createdById: ID
        searchString: String
        sortBy: PullRequestSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type PullRequestSearchResult {
        pageInfo: PageInfo!
        edges: [PullRequestEdge!]!
    }

    type PullRequestEdge {
        cursor: String!
        node: PullRequest!
    }

    extend type Query {
        pullRequest(input: FindByIdInput!): PullRequest
        pullRequests(input: PullRequestSearchInput!): PullRequestSearchResult!
    }

    extend type Mutation {
        pullRequestCreate(input: PullRequestCreateInput!): PullRequest!
        pullRequestUpdate(input: PullRequestUpdateInput!): PullRequest!
        pullRequestAccept(input: FindByIdInput!): PullRequest!
        pullRequestReject(input: FindByIdInput!): PullRequest!
    }
`

const objectType = 'PullRequest';
export const resolvers: {
    PullRequestSortBy: typeof PullRequestSortBy;
    PullRequestToObjectType: typeof PullRequestToObjectType;
    PullRequestStatus: typeof PullRequestStatus;
    PullRequestTo: UnionResolver;
    PullRequestFrom: UnionResolver;
    Query: {
        pullRequest: GQLEndpoint<FindByIdInput, FindOneResult<PullRequest>>;
        pullRequests: GQLEndpoint<PullRequestSearchInput, FindManyResult<PullRequest>>;
    },
    Mutation: {
        pullRequestCreate: GQLEndpoint<PullRequestCreateInput, CreateOneResult<PullRequest>>;
        pullRequestUpdate: GQLEndpoint<PullRequestUpdateInput, UpdateOneResult<PullRequest>>;
        pullRequestAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<PullRequest>>;
        pullRequestReject: GQLEndpoint<FindByIdInput, UpdateOneResult<PullRequest>>;
    }
} = {
    PullRequestSortBy,
    PullRequestToObjectType,
    PullRequestStatus,
    PullRequestTo: { __resolveType(obj: any) { return resolveUnion(obj) } },
    PullRequestFrom: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        pullRequest: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        pullRequests: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        pullRequestCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        pullRequestUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        pullRequestAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        pullRequestReject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        }
    }
}