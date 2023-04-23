import { TransferObjectType, TransferSortBy } from "@local/consts";
import { TransferStatus } from "@prisma/client";
import { gql } from "apollo-server-express";
import { readManyHelper, readOneHelper, updateHelper } from "../actions";
import { assertRequestFrom } from "../auth";
import { CustomError } from "../events";
import { rateLimit } from "../middleware";
import { TransferModel } from "../models";
import { resolveUnion } from "./resolvers";
export const typeDef = gql `
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
const objectType = "Transfer";
export const resolvers = {
    TransferSortBy,
    TransferObjectType,
    TransferStatus,
    TransferObject: { __resolveType(obj) { return resolveUnion(obj); } },
    Query: {
        transfer: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        transfers: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        transferRequestSend: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            const userData = assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer(prisma).requestSend(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req });
        },
        transferRequestReceive: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            const userData = assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer(prisma).requestReceive(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req });
        },
        transferUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        transferCancel: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        transferAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        transferDeny: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
//# sourceMappingURL=transfer.js.map