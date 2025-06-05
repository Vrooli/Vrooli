import { ScheduleSortBy, endpointsSchedule, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function scheduleSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchSchedule"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function scheduleSearchParams() {
    return toParams(scheduleSearchSchema(), endpointsSchedule, ScheduleSortBy, ScheduleSortBy.DateUpdatedDesc);
}
