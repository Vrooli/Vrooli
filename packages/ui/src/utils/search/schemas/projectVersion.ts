import { endpointGetProjectVersion, endpointGetProjectVersions, FormSchema, ProjectVersionSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const projectVersionSearchSchema = (): FormSchema => ({
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
});

export const projectVersionSearchParams = () => toParams(projectVersionSearchSchema(), endpointGetProjectVersions, endpointGetProjectVersion, ProjectVersionSortBy, ProjectVersionSortBy.DateCreatedDesc);
