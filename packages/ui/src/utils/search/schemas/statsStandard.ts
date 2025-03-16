import { endpointsStatsStandard, FormSchema, StatsStandardSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsStandardSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsStandard"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsStandardSearchParams() {
    return toParams(statsStandardSearchSchema(), endpointsStatsStandard, StatsStandardSortBy, StatsStandardSortBy.PeriodStartAsc);
}
