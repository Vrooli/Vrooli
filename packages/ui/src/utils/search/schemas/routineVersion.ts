import { RoutineVersionSortBy } from "@shared/consts";
import { routineVersionFindMany } from "api/generated/endpoints/routineVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, complexityContainer, complexityFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const routineVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRoutineVersion', lng),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        simplicityContainer,
        complexityContainer,
        votesRootContainer,
        bookmarksRootContainer,
        languagesContainer,
        tagsRootContainer,
    ],
    fields: [
        ...isCompleteWithRootFields,
        ...isLatestFields,
        ...simplicityFields,
        ...complexityFields,
        ...votesRootFields,
        ...bookmarksRootFields,
        ...languagesFields,
        ...tagsRootFields,
    ]
})

export const routineVersionSearchParams = (lng: string) => toParams(routineVersionSearchSchema(lng), routineVersionFindMany, RoutineVersionSortBy, RoutineVersionSortBy.DateCreatedDesc);