import { StatsSiteSortBy } from "@local/shared";
import { statsSiteFindMany } from "api/generated/endpoints/statsSite_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSiteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsSite'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsSiteSearchParams = () => toParams(statsSiteSearchSchema(), statsSiteFindMany, StatsSiteSortBy, StatsSiteSortBy.PeriodStartAsc);