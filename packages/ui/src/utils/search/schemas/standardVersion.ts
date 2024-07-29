import { endpointGetStandardVersion, endpointGetStandardVersions, FormSchema, StandardVersionSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const standardVersionSearchSchema = (): FormSchema => ({
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
});

export const standardVersionSearchParams = () => toParams(standardVersionSearchSchema(), endpointGetStandardVersions, endpointGetStandardVersion, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc);
