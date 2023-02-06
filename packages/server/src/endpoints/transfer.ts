import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from '../types';
import { FindByIdInput, TransferSortBy, TransferObjectType, Transfer, TransferSearchInput, TransferRequestSendInput, TransferRequestReceiveInput, TransferUpdateInput, TransferDenyInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { CustomError } from '../events';
import { resolveUnion } from './resolvers';
import { TransferStatus } from '@prisma/client';
import { assertRequestFrom } from '../auth';
import { TransferModel } from '../models';

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
        id: ID!
        objectType: TransferObjectType!
        objectConnect: ID!
        toOrganizationConnect: ID
        toUserConnect: ID
        message: String
    }
    input TransferRequestReceiveInput {
        id: ID!
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
`

const objectType = 'Transfer';
export const resolvers: {
    TransferSortBy: typeof TransferSortBy;
    TransferObjectType: typeof TransferObjectType;
    TransferStatus: typeof TransferStatus;
    TransferObject: UnionResolver;
    Query: {
        transfer: GQLEndpoint<FindByIdInput, FindOneResult<Transfer>>;
        transfers: GQLEndpoint<TransferSearchInput, FindManyResult<Transfer>>;
    },
    Mutation: {
        transferRequestSend: GQLEndpoint<TransferRequestSendInput, CreateOneResult<Transfer>>;
        transferRequestReceive: GQLEndpoint<TransferRequestReceiveInput, CreateOneResult<Transfer>>;
        transferUpdate: GQLEndpoint<TransferUpdateInput, UpdateOneResult<Transfer>>;
        transferCancel: GQLEndpoint<FindByIdInput, UpdateOneResult<Transfer>>;
        transferAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<Transfer>>;
        transferDeny: GQLEndpoint<TransferDenyInput, UpdateOneResult<Transfer>>;
    }
} = {
    TransferSortBy,
    TransferObjectType,
    TransferStatus,
    TransferObject: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        transfer: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        transfers: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        transferRequestSend: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            const userData = assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer(prisma).requestSend(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
        transferRequestReceive: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            const userData = assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer(prisma).requestReceive(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
        transferUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        transferCancel: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError('0000', 'NotImplemented', ['en']);
            // const transferId = await TransferModel.transfer(prisma).cancel(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
        transferAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError('0000', 'NotImplemented', ['en']);
            // const transferId = await TransferModel.transfer(prisma).accept(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
        transferDeny: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError('0000', 'NotImplemented', ['en']);
            // const transferId = await TransferModel.transfer(prisma).deny(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        }
    }
}