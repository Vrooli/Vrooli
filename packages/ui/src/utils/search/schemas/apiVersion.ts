import { ApiVersionSortBy } from "@shared/consts";
import { apiVersionFindMany } from "api/generated/endpoints/apiVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const apiVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchApiVersions', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const apiVersionSearchParams = (lng: string) => toParams(apiVersionSearchSchema(lng), apiVersionFindMany, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);