import { endpointsTransfer, FormSchema, TransferSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function transferSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchTransfer"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function transferSearchParams() {
    return toParams(transferSearchSchema(), endpointsTransfer, TransferSortBy, TransferSortBy.DateCreatedDesc);
}

