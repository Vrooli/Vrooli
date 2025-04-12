import { endpointsResourceList, FormSchema, ResourceListSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function resourceListSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchResourceList"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function resourceListSearchParams() {
    return toParams(resourceListSearchSchema(), endpointsResourceList, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc);
}
