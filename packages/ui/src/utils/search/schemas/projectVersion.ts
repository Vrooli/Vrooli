import { ProjectVersionSortBy } from "@shared/consts";
import { projectVersionFindMany } from "api/generated/endpoints/projectVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const projectVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectVersion', lng),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        votesRootContainer(lng),
        bookmarksRootContainer(lng),
        languagesContainer(lng),
        tagsRootContainer(lng),
    ],
    fields: [
        ...isCompleteWithRootFields(lng),
        ...isLatestFields(lng),
        ...votesRootFields(lng),
        ...bookmarksRootFields(lng),
        ...languagesFields(lng),
        ...tagsRootFields(lng),
    ]
})

export const projectVersionSearchParams = (lng: string) => toParams(projectVersionSearchSchema(lng), projectVersionFindMany, ProjectVersionSortBy, ProjectVersionSortBy.DateCreatedDesc)