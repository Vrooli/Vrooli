import { endpointGetUnionsRunProjectOrRunRoutines, RunProjectOrRunRoutineSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectOrRunRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRunProjectOrRunRoutine"),
    containers: [], //TODO
    fields: [], //TODO
});

export const runProjectOrRunRoutineSearchParams = () => toParams(runProjectOrRunRoutineSearchSchema(), endpointGetUnionsRunProjectOrRunRoutines, undefined, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);
