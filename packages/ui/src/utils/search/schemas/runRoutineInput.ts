import { RunRoutineInputSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { runRoutineInputFindMany } from "../../../api/generated/endpoints/runRoutineInput_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineInputSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRunRoutineInput"),
    containers: [], //TODO
    fields: [], //TODO
})

export const runRoutineInputSearchParams = () => toParams(runRoutineInputSearchSchema(), runRoutineInputFindMany, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc);