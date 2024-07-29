import { endpointGetSchedule, endpointGetSchedules, FormSchema, ScheduleSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const scheduleSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchSchedule"),
    containers: [], //TODO
    elements: [], //TODO
});

export const scheduleSearchParams = () => toParams(scheduleSearchSchema(), endpointGetSchedules, endpointGetSchedule, ScheduleSortBy, ScheduleSortBy.DateUpdatedDesc);
