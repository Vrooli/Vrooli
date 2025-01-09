import { endpointsPost, FormSchema, PostSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
