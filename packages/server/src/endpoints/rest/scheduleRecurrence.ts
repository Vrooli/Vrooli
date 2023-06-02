import { scheduleRecurrence_create, scheduleRecurrence_findMany, scheduleRecurrence_findOne, scheduleRecurrence_update } from "@local/shared";
import { ScheduleRecurrenceEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ScheduleRecurrenceRest = setupRoutes({
    "/scheduleRecurrence/:id": {
        get: [ScheduleRecurrenceEndpoints.Query.scheduleRecurrence, scheduleRecurrence_findOne],
        put: [ScheduleRecurrenceEndpoints.Mutation.scheduleRecurrenceUpdate, scheduleRecurrence_update],
    },
    "/scheduleRecurrences": {
        get: [ScheduleRecurrenceEndpoints.Query.scheduleRecurrences, scheduleRecurrence_findMany],
    },
    "/scheduleRecurrence": {
        post: [ScheduleRecurrenceEndpoints.Mutation.scheduleRecurrenceCreate, scheduleRecurrence_create],
    },
});
