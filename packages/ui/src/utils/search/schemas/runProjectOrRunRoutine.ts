import { RunProjectOrRunRoutineSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { runProjectOrRunRoutineFindMany } from "../../../api/generated/endpoints/runProjectOrRunRoutine_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectOrRunRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRunProjectOrRunRoutine"),
    containers: [], //TODO
    fields: [], //TODO
});

export const runProjectOrRunRoutineSearchParams = () => toParams(runProjectOrRunRoutineSearchSchema(), runProjectOrRunRoutineFindMany, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);
