import { StatsSiteSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { statsSiteFindMany } from "../../../api/generated/endpoints/statsSite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSiteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsSite"),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsSiteSearchParams = () => toParams(statsSiteSearchSchema(), statsSiteFindMany, StatsSiteSortBy, StatsSiteSortBy.PeriodStartAsc);