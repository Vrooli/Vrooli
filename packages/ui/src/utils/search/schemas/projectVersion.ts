import { endpointsProjectVersion, FormSchema, ProjectVersionSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export function projectVersionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchProjectVersion"),
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

export function projectVersionSearchParams() {
    return toParams(projectVersionSearchSchema(), endpointsProjectVersion, ProjectVersionSortBy, ProjectVersionSortBy.DateCreatedDesc);
}
