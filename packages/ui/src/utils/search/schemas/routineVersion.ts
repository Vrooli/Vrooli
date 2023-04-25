import { RoutineVersionSortBy } from "@local/shared";
import { routineVersionFindMany } from "api/generated/endpoints/routineVersion_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, complexityContainer, complexityFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const routineVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutineVersion'),
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
    fields: [
        ...isCompleteWithRootFields(),
        ...isLatestFields(),
        ...simplicityFields(),
        ...complexityFields(),
        ...votesRootFields(),
        ...bookmarksRootFields(),
        ...languagesFields(),
        ...tagsRootFields(),
    ]
})

export const routineVersionSearchParams = () => toParams(routineVersionSearchSchema(), routineVersionFindMany, RoutineVersionSortBy, RoutineVersionSortBy.DateCreatedDesc);