import { endpointsStatsProject, FormSchema, StatsProjectSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsProjectSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsProject"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsProjectSearchParams() {
    return toParams(statsProjectSearchSchema(), endpointsStatsProject, StatsProjectSortBy, StatsProjectSortBy.PeriodStartAsc);
}
