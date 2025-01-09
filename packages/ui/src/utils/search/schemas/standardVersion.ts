import { endpointsStandardVersion, FormSchema, StandardVersionSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export function standardVersionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStandardVersion"),
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

export function standardVersionSearchParams() {
    return toParams(standardVersionSearchSchema(), endpointsStandardVersion, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc);
}
