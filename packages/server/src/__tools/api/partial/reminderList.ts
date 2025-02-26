import { ReminderList } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminderList: ApiPartial<ReminderList> = {
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        focusMode: async () => rel((await import("./focusMode.js")).focusMode, "list", { omit: "reminderList" }),
        reminders: async () => rel((await import("./reminder.js")).reminder, "full", { omit: "reminderList" }),
    },
};
