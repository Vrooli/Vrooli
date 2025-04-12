import { endpointsPost, FormSchema, PostSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function postSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchPost"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function postSearchParams() {
    return toParams(postSearchSchema(), endpointsPost, PostSortBy, PostSortBy.DateCreatedDesc);
}
