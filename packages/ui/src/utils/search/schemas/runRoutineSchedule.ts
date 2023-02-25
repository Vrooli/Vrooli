import { RunRoutineScheduleSortBy } from "@shared/consts";
import { runRoutineScheduleFindMany } from "api/generated/endpoints/runRoutineSchedule";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineScheduleSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchRunRoutineSchedule'),
    containers: [], //TODO
    fields: [], //TODO
})

export const runRoutineScheduleSearchParams = () => toParams(runRoutineScheduleSearchSchema(), runRoutineScheduleFindMany, RunRoutineScheduleSortBy, RunRoutineScheduleSortBy.WindowStartAsc);