import { type FindByIdInput, type Transfer, type TransferDenyInput, type TransferRequestReceiveInput, type TransferRequestSendInput, type TransferSearchInput, type TransferSearchResult, type TransferUpdateInput } from "@vrooli/shared";
import { readOneHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { TransferModel } from "../../models/base/transfer.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

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
export const transfer: EndpointsTransfer = createStandardCrudEndpoints({
    objectType: "Transfer",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
    customEndpoints: {
        requestSendOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 100, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer().requestSend(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, req });
        },
        requestReceiveOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 100, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            const transferId = await TransferModel.transfer().requestReceive(info, input, userData);
            return readOneHelper({ info, input: { id: transferId }, objectType, req });
        },
        cancelOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented");
            // const transferId = await TransferModel.transfer().cancel(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, req })
        },
        acceptOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented");
            // const transferId = await TransferModel.transfer().accept(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, req })
        },
        denyOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 250, req });
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            throw new CustomError("0000", "NotImplemented");
            // const transferId = await TransferModel.transfer().deny(info, input, userData);
            // return readOneHelper({ info, input: { id: transferId }, objectType, req })
        },
    },
});
