import { ProjectSortBy } from "@shared/consts";
import { projectFindMany } from "api/generated/endpoints/project_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const projectSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchProject'),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ]
})

export const projectSearchParams = () => toParams(projectSearchSchema(), projectFindMany, ProjectSortBy, ProjectSortBy.ScoreDesc)