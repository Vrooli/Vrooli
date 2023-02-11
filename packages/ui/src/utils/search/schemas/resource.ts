import { ResourceSortBy } from "@shared/consts";
import { resourceFindMany } from "api/generated/endpoints/resource";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchResources', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const resourceSearchParams = (lng: string) => toParams(resourceSearchSchema(lng), resourceFindMany, ResourceSortBy, ResourceSortBy.DateCreatedDesc)