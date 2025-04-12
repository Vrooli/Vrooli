import { endpointsTag, FormSchema, TagSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function tagSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchTag"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function tagSearchParams() {
    return toParams(tagSearchSchema(), endpointsTag, TagSortBy, TagSortBy.EmbedTopDesc);
}
