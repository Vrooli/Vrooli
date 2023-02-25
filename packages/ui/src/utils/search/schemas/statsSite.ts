import { StatsSiteSortBy } from "@shared/consts";
import { statsSiteFindMany } from "api/generated/endpoints/statsSite";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSiteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsSite'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsSiteSearchParams = () => toParams(statsSiteSearchSchema(), statsSiteFindMany, StatsSiteSortBy, StatsSiteSortBy.DateUpdatedDesc);