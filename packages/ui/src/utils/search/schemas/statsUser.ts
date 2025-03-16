import { endpointsStatsUser, FormSchema, StatsUserSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsUserSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsUser"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsUserSearchParams() {
    return toParams(statsUserSearchSchema(), endpointsStatsUser, StatsUserSortBy, StatsUserSortBy.PeriodStartAsc);
}
