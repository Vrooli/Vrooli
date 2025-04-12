import { endpointsStatsCode, FormSchema, StatsCodeSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsCodeSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsCode"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsCodeSearchParams() {
    return toParams(statsCodeSearchSchema(), endpointsStatsCode, StatsCodeSortBy, StatsCodeSortBy.PeriodStartAsc);
}
