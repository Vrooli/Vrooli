import { FindByIdInput, Transfer, TransferDenyInput, TransferRequestReceiveInput, TransferRequestSendInput, TransferSearchInput, TransferSearchResult, TransferUpdateInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { TransferModel } from "../../models/base/transfer";
import { ApiEndpoint } from "../../types";

export type EndpointsTransfer = {
    findOne: ApiEndpoint<FindByIdInput, Transfer>;
    findMany: ApiEndpoint<TransferSearchInput, TransferSearchResult>;
    requestSendOne: ApiEndpoint<TransferRequestSendInput, Transfer>;
    requestReceiveOne: ApiEndpoint<TransferRequestReceiveInput, Transfer>;
    updateOne: ApiEndpoint<TransferUpdateInput, Transfer>;
    cancelOne: ApiEndpoint<FindByIdInput, Transfer>;
    acceptOne: ApiEndpoint<FindByIdInput, Transfer>;
    denyOne: ApiEndpoint<TransferDenyInput, Transfer>;
}

const objectType = "Transfer";
export const transfer: EndpointsTransfer = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    requestSendOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const transferId = await TransferModel.transfer().requestSend(info, input, userData);
        return readOneHelper({ info, input: { id: transferId }, objectType, req });
    },
    requestReceiveOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const transferId = await TransferModel.transfer().requestReceive(info, input, userData);
        return readOneHelper({ info, input: { id: transferId }, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    cancelOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        throw new CustomError("0000", "NotImplemented");
        // const transferId = await TransferModel.transfer().cancel(info, input, userData);
        // return readOneHelper({ info, input: { id: transferId }, objectType, req })
    },
    acceptOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        throw new CustomError("0000", "NotImplemented");
        // const transferId = await TransferModel.transfer().accept(info, input, userData);
        // return readOneHelper({ info, input: { id: transferId }, objectType, req })
    },
    denyOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        throw new CustomError("0000", "NotImplemented");
        // const transferId = await TransferModel.transfer().deny(info, input, userData);
        // return readOneHelper({ info, input: { id: transferId }, objectType, req })
    },
};
