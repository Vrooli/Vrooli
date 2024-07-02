import { endpointGetRoutineVersion, endpointGetRoutineVersions, RoutineVersionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, complexityContainer, complexityFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const routineVersionSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchRoutineVersion"),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        simplicityContainer(),
        complexityContainer(),
        votesRootContainer(),
        bookmarksRootContainer(),
        languagesContainer(),
        tagsRootContainer(),
    ],
    elements: [
        ...isCompleteWithRootFields(),
        ...isLatestFields(),
        ...simplicityFields(),
        ...complexityFields(),
        ...votesRootFields(),
        ...bookmarksRootFields(),
        ...languagesFields(),
        ...tagsRootFields(),
    ],
});

export const routineVersionSearchParams = () => toParams(routineVersionSearchSchema(), endpointGetRoutineVersions, endpointGetRoutineVersion, RoutineVersionSortBy, RoutineVersionSortBy.DateCreatedDesc);
