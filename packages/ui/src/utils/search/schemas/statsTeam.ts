import { endpointGetStatsTeam, StatsTeamSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsTeamSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsTeam"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsTeamSearchParams = () => toParams(statsTeamSearchSchema(), endpointGetStatsTeam, undefined, StatsTeamSortBy, StatsTeamSortBy.PeriodStartAsc);
