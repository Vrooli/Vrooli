import { CodeVersionSortBy, FormSchema, endpointsCodeVersion } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common.js";

export function codeVersionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchCodeVersion"),
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

export function codeVersionSearchParams() {
    return toParams(codeVersionSearchSchema(), endpointsCodeVersion, CodeVersionSortBy, CodeVersionSortBy.DateCreatedDesc);
}
