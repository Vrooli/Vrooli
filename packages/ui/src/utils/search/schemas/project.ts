import { ProjectSortBy } from "@shared/consts";
import { projectFindMany } from "api/generated/endpoints/project";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const projectSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProject', lng),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(lng),
        bookmarksContainer(lng),
        languagesVersionContainer(lng),
        tagsContainer(lng),
    ],
    fields: [
        ...hasCompleteVersionFields(lng),
        ...votesFields(lng),
        ...bookmarksFields(lng),
        ...languagesVersionFields(lng),
        ...tagsFields(lng),
    ]
})

export const projectSearchParams = (lng: string) => toParams(projectSearchSchema(lng), projectFindMany, ProjectSortBy, ProjectSortBy.ScoreDesc)