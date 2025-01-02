import { FindByIdInput, Transfer, TransferDenyInput, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { TransferModel } from "../../models/base/transfer";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsTransfer = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<Transfer>>;
    findMany: ApiEndpoint<TransferSearchInput, FindManyResult<Transfer>>;
    requestSendOne: ApiEndpoint<TransferRequestSendInput, CreateOneResult<Transfer>>;
    requestReceiveOne: ApiEndpoint<TransferRequestReceiveInput, CreateOneResult<Transfer>>;
    updateOne: ApiEndpoint<TransferUpdateInput, UpdateOneResult<Transfer>>;
    cancelOne: ApiEndpoint<FindByIdInput, UpdateOneResult<Transfer>>;
    acceptOne: ApiEndpoint<FindByIdInput, UpdateOneResult<Transfer>>;
    denyOne: ApiEndpoint<TransferDenyInput, UpdateOneResult<Transfer>>;
}

const objectType = "Transfer";
export const transfer: EndpointsTransfer = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    requestSendOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const transferId = await TransferModel.transfer().requestSend(info, input, userData);
        return readOneHelper({ info, input: { id: transferId }, objectType, req });
    },
    requestReceiveOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const transferId = await TransferModel.transfer().requestReceive(info, input, userData);
        return readOneHelper({ info, input: { id: transferId }, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    cancelOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        throw new CustomError("0000", "NotImplemented");
        // const transferId = await TransferModel.transfer().cancel(info, input, userData);
        // return readOneHelper({ info, input: { id: transferId }, objectType, req })
    },
    acceptOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        throw new CustomError("0000", "NotImplemented");
        // const transferId = await TransferModel.transfer().accept(info, input, userData);
        // return readOneHelper({ info, input: { id: transferId }, objectType, req })
    },
    denyOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        throw new CustomError("0000", "NotImplemented");
        // const transferId = await TransferModel.transfer().deny(info, input, userData);
        // return readOneHelper({ info, input: { id: transferId }, objectType, req })
    },
};
