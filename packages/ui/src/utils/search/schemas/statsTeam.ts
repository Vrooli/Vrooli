import { endpointsStatsTeam, FormSchema, StatsTeamSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
