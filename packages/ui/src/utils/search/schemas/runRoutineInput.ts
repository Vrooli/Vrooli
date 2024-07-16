import { endpointGetRunRoutineInputs, RunRoutineInputSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineInputSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchRunRoutineInput"),
    containers: [], //TODO
    elements: [], //TODO
});

export const runRoutineInputSearchParams = () => toParams(runRoutineInputSearchSchema(), endpointGetRunRoutineInputs, undefined, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc);
