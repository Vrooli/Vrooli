import { endpointGetUnionsRunProjectOrRunRoutines, RunProjectOrRunRoutineSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectOrRunRoutineSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchRunProjectOrRunRoutine"),
    containers: [], //TODO
    elements: [], //TODO
});

export const runProjectOrRunRoutineSearchParams = () => toParams(runProjectOrRunRoutineSearchSchema(), endpointGetUnionsRunProjectOrRunRoutines, undefined, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);
