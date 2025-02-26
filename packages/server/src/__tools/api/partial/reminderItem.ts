import { ReminderItem } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminderItem: ApiPartial<ReminderItem> = {
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        index: true,
        isComplete: true,
        reminder: async () => rel((await import("./reminder.js")).reminder, "full", { omit: "reminderItems" }),
    },
};
