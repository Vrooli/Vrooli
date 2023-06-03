import { StatsSiteSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSiteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsSite"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsSiteSearchParams = () => toParams(statsSiteSearchSchema(), "/stats/site", StatsSiteSortBy, StatsSiteSortBy.PeriodStartAsc);
