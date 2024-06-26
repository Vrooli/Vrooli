import { endpointGetRunRoutineInputs, RunRoutineInputSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineInputSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRunRoutineInput"),
    containers: [], //TODO
    fields: [], //TODO
});

export const runRoutineInputSearchParams = () => toParams(runRoutineInputSearchSchema(), endpointGetRunRoutineInputs, undefined, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc);
