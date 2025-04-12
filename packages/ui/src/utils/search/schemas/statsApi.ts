import { endpointsStatsApi, FormSchema, StatsApiSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
