import { ApiVersionSortBy, FormSchema, endpointsApiVersion } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common.js";

export function apiVersionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchApiVersion"),
        containers: [
            isCompleteWithRootContainer,
            isLatestContainer,
            votesRootContainer(),
            bookmarksRootContainer(),
            languagesContainer(),
            tagsRootContainer(),
        ],
        elements: [
            ...isCompleteWithRootFields(),
            ...isLatestFields(),
            ...votesRootFields(),
            ...bookmarksRootFields(),
            ...languagesFields(),
            ...tagsRootFields(),
        ],
    };
}

export function apiVersionSearchParams() {
    return toParams(apiVersionSearchSchema(), endpointsApiVersion, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);
}

