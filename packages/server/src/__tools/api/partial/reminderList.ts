import { ReminderList } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminderList: ApiPartial<ReminderList> = {
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        reminders: async () => rel((await import("./reminder.js")).reminder, "full", { omit: "reminderList" }),
    },
};
