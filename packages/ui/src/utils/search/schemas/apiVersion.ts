import { ApiVersionSortBy, endpointGetApiVersion, endpointGetApiVersions } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const apiVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchApiVersion"),
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

export const apiVersionSearchParams = () => toParams(apiVersionSearchSchema(), endpointGetApiVersions, endpointGetApiVersion, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);
