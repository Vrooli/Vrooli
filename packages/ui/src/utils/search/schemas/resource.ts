import { endpointsResource, FormSchema, ResourceSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
