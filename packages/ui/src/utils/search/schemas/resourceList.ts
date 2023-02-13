import { ResourceListSortBy } from "@shared/consts";
import { resourceListFindMany } from "api/generated/endpoints/resourceList";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceListSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchResourceList', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const resourceListSearchParams = (lng: string) => toParams(resourceListSearchSchema(lng), resourceListFindMany, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc)