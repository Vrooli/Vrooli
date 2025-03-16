import { endpointsResource, FormSchema, ResourceSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function resourceSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchResource"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function resourceSearchParams() {
    return toParams(resourceSearchSchema(), endpointsResource, ResourceSortBy, ResourceSortBy.DateCreatedDesc);
}
