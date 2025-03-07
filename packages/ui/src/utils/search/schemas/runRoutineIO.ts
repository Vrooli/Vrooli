import { endpointsRunRoutineIO, FormSchema, RunRoutineIOSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function runRoutineIOSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRunRoutineIO"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function runRoutineIOSearchParams() {
    return toParams(runRoutineIOSearchSchema(), endpointsRunRoutineIO, RunRoutineIOSortBy, RunRoutineIOSortBy.DateCreatedDesc);
}
