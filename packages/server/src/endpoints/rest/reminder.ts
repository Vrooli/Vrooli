import { reminder_create, reminder_findMany, reminder_findOne, reminder_update } from "../generated";
import { ReminderEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ReminderRest = setupRoutes({
    "/reminder/:id": {
        get: [ReminderEndpoints.Query.reminder, reminder_findOne],
        put: [ReminderEndpoints.Mutation.reminderUpdate, reminder_update],
    },
    "/reminders": {
        get: [ReminderEndpoints.Query.reminders, reminder_findMany],
    },
    "/reminder": {
        post: [ReminderEndpoints.Mutation.reminderCreate, reminder_create],
    },
});
