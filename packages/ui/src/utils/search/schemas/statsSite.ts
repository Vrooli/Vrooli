import { StatsSiteSortBy } from "@shared/consts";
import { statsSiteFindMany } from "api/generated/endpoints/statsSite";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSiteSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsSite', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsSiteSearchParams = (lng: string) => toParams(statsSiteSearchSchema(lng), statsSiteFindMany, StatsSiteSortBy, StatsSiteSortBy.DateUpdatedDesc);