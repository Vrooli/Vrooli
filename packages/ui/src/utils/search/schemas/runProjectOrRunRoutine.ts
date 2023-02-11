import { RunProjectOrRunRoutineSortBy } from "@shared/consts";
import { runProjectOrRunRoutineFindMany } from "api/generated/endpoints/runProjectOrRunRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectOrRunRoutineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRunProjectsOrRunRoutines', lng),
    containers: [], //TODO
    fields: [] //TODO
})

export const runProjectOrRunRoutineSearchParams = (lng: string) => toParams(runProjectOrRunRoutineSearchSchema(lng), runProjectOrRunRoutineFindMany, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);