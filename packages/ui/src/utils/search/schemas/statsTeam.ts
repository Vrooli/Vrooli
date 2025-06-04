import { endpointsStatsTeam, type FormSchema, StatsTeamSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsTeamSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsTeam"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsTeamSearchParams() {
    return toParams(statsTeamSearchSchema(), endpointsStatsTeam, StatsTeamSortBy, StatsTeamSortBy.PeriodStartAsc);
}
