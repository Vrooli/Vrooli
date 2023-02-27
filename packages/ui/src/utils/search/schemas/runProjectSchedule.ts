import { RunProjectScheduleSortBy } from "@shared/consts";
import { runProjectScheduleFindMany } from "api/generated/endpoints/runProjectSchedule_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectScheduleSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchRunProjectSchedule'),
    containers: [], //TODO
    fields: [], //TODO
})

export const runProjectScheduleSearchParams = () => toParams(runProjectScheduleSearchSchema(), runProjectScheduleFindMany, RunProjectScheduleSortBy, RunProjectScheduleSortBy.WindowStartAsc);