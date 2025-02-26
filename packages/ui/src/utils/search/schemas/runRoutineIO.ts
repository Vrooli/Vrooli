import { endpointsRunRoutineIO, FormSchema, RunRoutineIOSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function runRoutineInputSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRunRoutineInput"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function runRoutineInputSearchParams() {
    return toParams(runRoutineInputSearchSchema(), endpointsRunRoutineIO, RunRoutineIOSortBy, RunRoutineIOSortBy.DateCreatedDesc);
}
