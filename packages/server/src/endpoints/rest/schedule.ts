import { schedule_create, schedule_findMany, schedule_findOne, schedule_update } from "../generated";
import { ScheduleEndpoints } from "../logic/schedule";
import { setupRoutes } from "./base";

export const ScheduleRest = setupRoutes({
    "/schedule/:id": {
        get: [ScheduleEndpoints.Query.schedule, schedule_findOne],
        put: [ScheduleEndpoints.Mutation.scheduleUpdate, schedule_update],
    },
    "/schedules": {
        get: [ScheduleEndpoints.Query.schedules, schedule_findMany],
    },
    "/schedule": {
        post: [ScheduleEndpoints.Mutation.scheduleCreate, schedule_create],
    },
});
