import { RunRoutineInputSortBy } from "@shared/consts";
import { runRoutineInputFindMany } from "api/generated/endpoints/runRoutineInput";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineInputSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRunRoutineInput', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const runRoutineInputSearchParams = (lng: string) => toParams(runRoutineInputSearchSchema(lng), runRoutineInputFindMany, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc);