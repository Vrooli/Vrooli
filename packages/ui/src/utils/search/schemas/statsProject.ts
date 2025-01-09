import { endpointsStatsProject, FormSchema, StatsProjectSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
