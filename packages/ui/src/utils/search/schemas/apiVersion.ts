import { ApiVersionSortBy, FormSchema, endpointGetApiVersion, endpointGetApiVersions } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

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
    return toParams(apiVersionSearchSchema(), endpointGetApiVersions, endpointGetApiVersion, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);
}

