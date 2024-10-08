import { endpointGetStatsCode, FormSchema, StatsCodeSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsCodeSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsCode"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsCodeSearchParams = () => toParams(statsCodeSearchSchema(), endpointGetStatsCode, undefined, StatsCodeSortBy, StatsCodeSortBy.PeriodStartAsc);
