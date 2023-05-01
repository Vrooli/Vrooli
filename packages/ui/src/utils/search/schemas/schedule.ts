import { ScheduleSortBy } from "@local/shared";
import { scheduleFindMany } from "api/generated/endpoints/schedule_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const scheduleSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchSchedule"),
    containers: [], //TODO
    fields: [], //TODO
});

export const scheduleSearchParams = () => toParams(scheduleSearchSchema(), scheduleFindMany, ScheduleSortBy, ScheduleSortBy.DateUpdatedDesc);
