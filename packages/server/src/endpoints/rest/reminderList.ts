import { reminderList_create, reminderList_update } from "../generated";
import { ReminderListEndpoints } from "../logic/reminderList";
import { setupRoutes } from "./base";

export const ReminderListRest = setupRoutes({
    "/reminderList": {
        post: [ReminderListEndpoints.Mutation.reminderListCreate, reminderList_create],
    },
    "/reminderList/:id": {
        put: [ReminderListEndpoints.Mutation.reminderListUpdate, reminderList_update],
    },
});
