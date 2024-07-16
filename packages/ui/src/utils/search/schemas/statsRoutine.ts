import { endpointGetStatsRoutine, StatsRoutineSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsRoutineSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsRoutine"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsRoutineSearchParams = () => toParams(statsRoutineSearchSchema(), endpointGetStatsRoutine, undefined, StatsRoutineSortBy, StatsRoutineSortBy.PeriodStartAsc);
