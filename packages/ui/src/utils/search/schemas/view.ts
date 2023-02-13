import { ViewSortBy } from "@shared/consts";
import { viewFindMany } from "api/generated/endpoints/view";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const viewSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchView', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const viewSearchParams = (lng: string) => toParams(viewSearchSchema(lng), viewFindMany, ViewSortBy, ViewSortBy.LastViewedDesc)