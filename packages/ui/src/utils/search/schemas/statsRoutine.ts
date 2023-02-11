import { StatsRoutineSortBy } from "@shared/consts";
import { statsRoutineFindMany } from "api/generated/endpoints/statsRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsRoutineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsRoutine', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsRoutineSearchParams = (lng: string) => toParams(statsRoutineSearchSchema(lng), statsRoutineFindMany, StatsRoutineSortBy, StatsRoutineSortBy.DateUpdatedDesc);