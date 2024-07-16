import { endpointGetStatsProject, StatsProjectSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsProjectSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsProject"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsProjectSearchParams = () => toParams(statsProjectSearchSchema(), endpointGetStatsProject, undefined, StatsProjectSortBy, StatsProjectSortBy.PeriodStartAsc);
