import { endpointGetStatsTeam, StatsTeamSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsTeamSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsTeam"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsTeamSearchParams = () => toParams(statsTeamSearchSchema(), endpointGetStatsTeam, undefined, StatsTeamSortBy, StatsTeamSortBy.PeriodStartAsc);
