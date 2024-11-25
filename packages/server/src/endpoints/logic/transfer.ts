import { FindByIdInput, Transfer, TransferDenyInput, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { TransferModel } from "../../models/base/transfer";
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
        transfer: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        transfers: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        transferRequestSend: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer().requestSend(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, req });
        },
        transferRequestReceive: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer().requestReceive(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, req });
        },
        transferUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        transferCancel: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented");
            // const transferId = await TransferModel.transfer().cancel(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, req })
        },
        transferAccept: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented");
            // const transferId = await TransferModel.transfer().accept(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, req })
        },
        transferDeny: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented");
            // const transferId = await TransferModel.transfer().deny(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, req })
        },
    },
};
