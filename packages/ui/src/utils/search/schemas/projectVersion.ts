import { ProjectVersionSortBy } from "@shared/consts";
import { projectVersionFindMany } from "api/generated/endpoints/projectVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const projectVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectVersions', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const projectVersionSearchParams = (lng: string) => toParams(projectVersionSearchSchema(lng), projectVersionFindMany, ProjectVersionSortBy, ProjectVersionSortBy.DateCreatedDesc)