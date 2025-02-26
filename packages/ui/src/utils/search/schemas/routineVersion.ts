import { endpointsRoutineVersion, FormSchema, RoutineVersionSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksRootContainer, bookmarksRootFields, complexityContainer, complexityFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common.js";

export function routineVersionSearchSchema(): FormSchema {
    return {
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
    };
}

export function routineVersionSearchParams() { return toParams(routineVersionSearchSchema(), endpointsRoutineVersion, RoutineVersionSortBy, RoutineVersionSortBy.DateCreatedDesc); }
