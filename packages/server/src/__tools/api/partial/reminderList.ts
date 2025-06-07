import { type ReminderList } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminderList: ApiPartial<ReminderList> = {
    full: {
        id: true,
        createdAt: true,
        updatedAt: true,
        reminders: async () => rel((await import("./reminder.js")).reminder, "full", { omit: "reminderList" }),
    },
};
