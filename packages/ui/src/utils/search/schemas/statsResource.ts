import { StatsResourceSortBy, endpointsStatsResource, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsResourceSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsResource"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsResourceSearchParams() {
    return toParams(statsResourceSearchSchema(), endpointsStatsResource, StatsResourceSortBy, StatsResourceSortBy.PeriodStartAsc);
}
