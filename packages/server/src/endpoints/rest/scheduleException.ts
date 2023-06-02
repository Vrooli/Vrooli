import { scheduleException_create, scheduleException_findMany, scheduleException_findOne, scheduleException_update } from "@local/shared";
import { ScheduleExceptionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ScheduleExceptionRest = setupRoutes({
    "/scheduleException/:id": {
        get: [ScheduleExceptionEndpoints.Query.scheduleException, scheduleException_findOne],
        put: [ScheduleExceptionEndpoints.Mutation.scheduleExceptionUpdate, scheduleException_update],
    },
    "/scheduleExceptions": {
        get: [ScheduleExceptionEndpoints.Query.scheduleExceptions, scheduleException_findMany],
    },
    "/scheduleException": {
        post: [ScheduleExceptionEndpoints.Mutation.scheduleExceptionCreate, scheduleException_create],
    },
});
