import { CodeVersionSortBy, endpointGetCodeVersion, endpointGetCodeVersions } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const codeVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchCodeVersion"),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        votesRootContainer(),
        bookmarksRootContainer(),
        languagesContainer(),
        tagsRootContainer(),
    ],
    fields: [
        ...isCompleteWithRootFields(),
        ...isLatestFields(),
        ...votesRootFields(),
        ...bookmarksRootFields(),
        ...languagesFields(),
        ...tagsRootFields(),
    ],
});

export const codeVersionSearchParams = () => toParams(codeVersionSearchSchema(), endpointGetCodeVersions, endpointGetCodeVersion, CodeVersionSortBy, CodeVersionSortBy.DateCreatedDesc);
