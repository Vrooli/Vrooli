import { ResourceListSortBy } from "@local/shared";
import { resourceListFindMany } from "api/generated/endpoints/resourceList_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceListSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchResourceList'),
    containers: [], //TODO
    fields: [], //TODO
})

export const resourceListSearchParams = () => toParams(resourceListSearchSchema(), resourceListFindMany, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc)