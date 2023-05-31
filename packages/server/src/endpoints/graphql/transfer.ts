import { TransferObjectType, TransferSortBy } from "@local/shared";
import { TransferStatus } from "@prisma/client";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { EndpointsTransfer, TransferEndpoints } from "../logic";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
    enum TransferSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum TransferObjectType {
        Api
        Note
        Project
        Routine
        SmartContract
        Standard
    }

    enum TransferStatus {
        Accepted
        Denied
        Pending
    }

    union TransferObject = Api | Note | Project | Routine | SmartContract | Standard

    input TransferRequestSendInput {
        objectType: TransferObjectType!
        objectConnect: ID!
        toOrganizationConnect: ID
        toUserConnect: ID
        message: String
    }
    input TransferRequestReceiveInput {
        objectType: TransferObjectType!
        objectConnect: ID!
        toOrganizationConnect: ID # If not set, uses your userId
        message: String
    }
    input TransferUpdateInput {
        id: ID!
        message: String
    }
    type Transfer {
        id: ID!
        created_at: Date!
        updated_at: Date!
        mergedOrRejectedAt: Date
        status: TransferStatus!
        fromOwner: Owner
        toOwner: Owner
        object: TransferObject!
        you: TransferYou!
    }

    type TransferYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input TransferSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: TransferStatus
        fromOrganizationId: ID # If not set, uses your userId
        toOrganizationId: ID
        toUserId: ID
        searchString: String
        sortBy: TransferSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
        apiId: ID
        noteId: ID
        projectId: ID
        routineId: ID
        smartContractId: ID
        standardId: ID
    }

    type TransferSearchResult {
        pageInfo: PageInfo!
        edges: [TransferEdge!]!
    }

    type TransferEdge {
        cursor: String!
        node: Transfer!
    }

    input TransferDenyInput {
        id: ID!
        reason: String
    }

    extend type Query {
        transfer(input: FindByIdInput!): Transfer
        transfers(input: TransferSearchInput!): TransferSearchResult!
    }

    extend type Mutation {
        transferRequestSend(input: TransferRequestSendInput!): Transfer!
        transferRequestReceive(input: TransferRequestReceiveInput!): Transfer!
        transferUpdate(input: TransferUpdateInput!): Transfer!
        transferCancel(input: FindByIdInput!): Transfer!
        transferAccept(input: FindByIdInput!): Transfer!
        transferDeny(input: TransferDenyInput!): Transfer!
    }
`;

export const resolvers: {
    TransferSortBy: typeof TransferSortBy;
    TransferObjectType: typeof TransferObjectType;
    TransferStatus: typeof TransferStatus;
    TransferObject: UnionResolver;
    Query: EndpointsTransfer["Query"];
    Mutation: EndpointsTransfer["Mutation"];
} = {
    TransferSortBy,
    TransferObjectType,
    TransferStatus,
    TransferObject: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...TransferEndpoints,
};
