import { endpointGetStatsUser, StatsUserSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsUserSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsUser"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsUserSearchParams = () => toParams(statsUserSearchSchema(), endpointGetStatsUser, undefined, StatsUserSortBy, StatsUserSortBy.PeriodStartAsc);
