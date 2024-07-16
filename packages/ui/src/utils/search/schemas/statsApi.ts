import { endpointGetStatsApi, StatsApiSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsApiSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsApi"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsApiSearchParams = () => toParams(statsApiSearchSchema(), endpointGetStatsApi, undefined, StatsApiSortBy, StatsApiSortBy.PeriodStartAsc);
