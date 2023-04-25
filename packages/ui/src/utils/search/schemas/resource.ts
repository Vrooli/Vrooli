import { ResourceSortBy } from "@local/shared";
import { resourceFindMany } from "api/generated/endpoints/resource_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchResource'),
    containers: [], //TODO
    fields: [], //TODO
})

export const resourceSearchParams = () => toParams(resourceSearchSchema(), resourceFindMany, ResourceSortBy, ResourceSortBy.DateCreatedDesc)