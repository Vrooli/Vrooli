import { endpointGetStatsTeam, FormSchema, StatsTeamSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsTeamSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsTeam"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsTeamSearchParams = () => toParams(statsTeamSearchSchema(), endpointGetStatsTeam, undefined, StatsTeamSortBy, StatsTeamSortBy.PeriodStartAsc);
