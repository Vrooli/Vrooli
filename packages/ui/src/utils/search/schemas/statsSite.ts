import { endpointGetStatsSite, StatsSiteSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsSiteSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsSite"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsSiteSearchParams = () => toParams(statsSiteSearchSchema(), endpointGetStatsSite, undefined, StatsSiteSortBy, StatsSiteSortBy.PeriodStartAsc);
