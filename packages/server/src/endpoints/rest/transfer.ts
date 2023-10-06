import { transfer_accept, transfer_cancel, transfer_deny, transfer_findMany, transfer_findOne, transfer_requestReceive, transfer_requestSend, transfer_update } from "../generated";
import { TransferEndpoints } from "../logic/transfer";
import { setupRoutes } from "./base";

export const TransferRest = setupRoutes({
    "/transfer/:id": {
        get: [TransferEndpoints.Query.transfer, transfer_findOne],
        put: [TransferEndpoints.Mutation.transferUpdate, transfer_update],
    },
    "/transfers": {
        get: [TransferEndpoints.Query.transfers, transfer_findMany],
    },
    "/transfer/requestSend": {
        post: [TransferEndpoints.Mutation.transferRequestSend, transfer_requestSend],
    },
    "/transfer/requestReceive": {
        post: [TransferEndpoints.Mutation.transferRequestReceive, transfer_requestReceive],
    },
    "/transfer/:id/cancel": {
        put: [TransferEndpoints.Mutation.transferCancel, transfer_cancel],
    },
    "/transfer/:id/accept": {
        put: [TransferEndpoints.Mutation.transferAccept, transfer_accept],
    },
    "/transfer/deny": {
        put: [TransferEndpoints.Mutation.transferDeny, transfer_deny],
    },
});
