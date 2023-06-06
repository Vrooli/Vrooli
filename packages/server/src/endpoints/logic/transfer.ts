import { FindByIdInput, Transfer, TransferDenyInput, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { TransferModel } from "../../models/base";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsTransfer = {
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
}

const objectType = "Transfer";
export const TransferEndpoints: EndpointsTransfer = {
    Query: {
        transfer: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        transfers: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        transferRequestSend: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            const userData = assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer(prisma).requestSend(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req });
        },
        transferRequestReceive: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            const userData = assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer(prisma).requestReceive(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req });
        },
        transferUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        transferCancel: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented", ["en"]);
            // const transferId = await TransferModel.transfer(prisma).cancel(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
        transferAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented", ["en"]);
            // const transferId = await TransferModel.transfer(prisma).accept(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
        transferDeny: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            const userData = assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented", ["en"]);
            // const transferId = await TransferModel.transfer(prisma).deny(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, prisma, req })
        },
    },
};
