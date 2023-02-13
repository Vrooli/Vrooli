import { RunRoutineScheduleSortBy } from "@shared/consts";
import { runRoutineScheduleFindMany } from "api/generated/endpoints/runRoutineSchedule";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineScheduleSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRunRoutineSchedule', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const runRoutineScheduleSearchParams = (lng: string) => toParams(runRoutineScheduleSearchSchema(lng), runRoutineScheduleFindMany, RunRoutineScheduleSortBy, RunRoutineScheduleSortBy.WindowStartAsc);