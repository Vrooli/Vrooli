import { StatsRoutineSortBy } from "@shared/consts";
import { statsRoutineFindMany } from "api/generated/endpoints/statsRoutine_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsRoutine'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsRoutineSearchParams = () => toParams(statsRoutineSearchSchema(), statsRoutineFindMany, StatsRoutineSortBy, StatsRoutineSortBy.DateUpdatedDesc);