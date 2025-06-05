import { TagSortBy, endpointsTag, type FormSchema } from "@vrooli/shared";
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
