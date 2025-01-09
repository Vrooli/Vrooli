import { endpointsTag, FormSchema, TagSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
