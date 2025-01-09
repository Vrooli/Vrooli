import { endpointsRunRoutineInput, FormSchema, RunRoutineInputSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function runRoutineInputSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRunRoutineInput"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function runRoutineInputSearchParams() {
    return toParams(runRoutineInputSearchSchema(), endpointsRunRoutineInput, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc);
}
