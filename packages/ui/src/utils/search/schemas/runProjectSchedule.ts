import { RunProjectScheduleSortBy } from "@shared/consts";
import { runProjectScheduleFindMany } from "api/generated/endpoints/runProjectSchedule";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectScheduleSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRunProjectSchedule', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const runProjectScheduleSearchParams = (lng: string) => toParams(runProjectScheduleSearchSchema(lng), runProjectScheduleFindMany, RunProjectScheduleSortBy, RunProjectScheduleSortBy.WindowStartAsc);