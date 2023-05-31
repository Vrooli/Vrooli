
import { FindByIdInput, PullRequest, PullRequestCreateInput, PullRequestFromObjectType, PullRequestSearchInput, PullRequestSortBy, PullRequestStatus, PullRequestToObjectType, PullRequestUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from "../../types";
import { resolveUnion } from "./resolvers";

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

    enum PullRequestFromObjectType {
        ApiVersion
        NoteVersion
        ProjectVersion
        RoutineVersion
        SmartContractVersion
        StandardVersion
    }

    enum PullRequestStatus {
        Open
        Canceled
        Merged
        Rejected
    }

    union PullRequestTo = Api | Note | Project | Routine | SmartContract | Standard

    union PullRequestFrom = ApiVersion | NoteVersion | ProjectVersion | RoutineVersion | SmartContractVersion | StandardVersion

    input PullRequestCreateInput {
        id: ID!
        toObjectType: PullRequestToObjectType!
        toConnect: ID!
        fromObjectType: PullRequestFromObjectType!
        fromConnect: ID!
        translationsCreate: [PullRequestTranslationCreateInput!]
    }
    input PullRequestUpdateInput {
        id: ID!
        status: PullRequestStatus
        translationsDelete: [ID!]
        translationsCreate: [PullRequestTranslationCreateInput!]
        translationsUpdate: [PullRequestTranslationUpdateInput!]
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
        commentsCount: Int!
        translations: [CommentTranslation!]!
        translationsCount: Int!
        you: PullRequestYou!
    }

    input PullRequestTranslationCreateInput {
        id: ID!
        language: String!
        text: String!
    }
    input PullRequestTranslationUpdateInput {
        id: ID!
        language: String!
        text: String
    }
    type PullRequestTranslation {
        id: ID!
        language: String!
        text: String!
    }

    type PullRequestYou {
        canComment: Boolean!
        canDelete: Boolean!
        canReport: Boolean!
        canUpdate: Boolean!
    }

    input PullRequestSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isMergedOrRejected: Boolean
        translationLanguages: [String!]
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
`;

const objectType = "PullRequest";
export const resolvers: {
    PullRequestSortBy: typeof PullRequestSortBy;
    PullRequestToObjectType: typeof PullRequestToObjectType;
    PullRequestFromObjectType: typeof PullRequestFromObjectType;
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
    }
} = {
    PullRequestSortBy,
    PullRequestToObjectType,
    PullRequestFromObjectType,
    PullRequestStatus,
    PullRequestTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    PullRequestFrom: { __resolveType(obj: any) { return resolveUnion(obj); } },
    Query: {
        pullRequest: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        pullRequests: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        pullRequestCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        pullRequestUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
            // TODO make sure to set hasBeenClosedOrRejected to true if status is closed or rejected
            // TODO 2 permissions for this differ from normal objects. Some fields can be updated by creator, and some by owner of object the pull request is for. Probably need to make custom endpoints like for transfers
        },
    },
};
