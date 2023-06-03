import { StatsRoutineSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsRoutine"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsRoutineSearchParams = () => toParams(statsRoutineSearchSchema(), "/stats/routine", StatsRoutineSortBy, StatsRoutineSortBy.PeriodStartAsc);
