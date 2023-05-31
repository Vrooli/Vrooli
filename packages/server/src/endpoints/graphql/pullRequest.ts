
import { PullRequestFromObjectType, PullRequestSortBy, PullRequestStatus, PullRequestToObjectType } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { EndpointsPullRequest, PullRequestEndpoints } from "../logic";
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

export const resolvers: {
    PullRequestSortBy: typeof PullRequestSortBy;
    PullRequestToObjectType: typeof PullRequestToObjectType;
    PullRequestFromObjectType: typeof PullRequestFromObjectType;
    PullRequestStatus: typeof PullRequestStatus;
    PullRequestTo: UnionResolver;
    PullRequestFrom: UnionResolver;
    Query: EndpointsPullRequest["Query"];
    Mutation: EndpointsPullRequest["Mutation"];
} = {
    PullRequestSortBy,
    PullRequestToObjectType,
    PullRequestFromObjectType,
    PullRequestStatus,
    PullRequestTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    PullRequestFrom: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...PullRequestEndpoints,
};
