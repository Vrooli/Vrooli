import { RunProjectOrRunRoutineSortBy } from "@shared/consts";
import { runProjectOrRunRoutineFindMany } from "api/generated/endpoints/runProjectOrRunRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectOrRunRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchRunProjectOrRunRoutine'),
    containers: [], //TODO
    fields: [] //TODO
})

export const runProjectOrRunRoutineSearchParams = () => toParams(runProjectOrRunRoutineSearchSchema(), runProjectOrRunRoutineFindMany, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);