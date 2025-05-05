import { endpointsRunIO, FormSchema, RunIOSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function runIOSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRunIO"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function runIOSearchParams() {
    return toParams(runIOSearchSchema(), endpointsRunIO, RunIOSortBy, RunIOSortBy.DateCreatedDesc);
}
