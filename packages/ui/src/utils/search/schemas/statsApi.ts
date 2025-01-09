import { endpointsStatsApi, FormSchema, StatsApiSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function statsApiSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsApi"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsApiSearchParams() {
    return toParams(statsApiSearchSchema(), endpointsStatsApi, StatsApiSortBy, StatsApiSortBy.PeriodStartAsc);
}
