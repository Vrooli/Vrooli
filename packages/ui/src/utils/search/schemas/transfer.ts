import { TransferSortBy, endpointsTransfer, type FormSchema } from "@vrooli/shared";
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

