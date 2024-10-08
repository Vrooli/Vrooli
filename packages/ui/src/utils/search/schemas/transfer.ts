import { endpointGetTransfer, endpointGetTransfers, FormSchema, TransferSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function transferSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchTransfer"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function transferSearchParams() {
    return toParams(transferSearchSchema(), endpointGetTransfers, endpointGetTransfer, TransferSortBy, TransferSortBy.DateCreatedDesc);
}

