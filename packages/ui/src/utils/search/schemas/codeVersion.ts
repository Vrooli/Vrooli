import { CodeVersionSortBy, FormSchema, endpointGetCodeVersion, endpointGetCodeVersions } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const codeVersionSearchSchema = (): FormSchema => ({
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
});

export const codeVersionSearchParams = () => toParams(codeVersionSearchSchema(), endpointGetCodeVersions, endpointGetCodeVersion, CodeVersionSortBy, CodeVersionSortBy.DateCreatedDesc);
