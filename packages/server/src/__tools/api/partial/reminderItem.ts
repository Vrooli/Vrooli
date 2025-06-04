import { type ReminderItem } from "@local/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminderItem: ApiPartial<ReminderItem> = {
    full: {
        id: true,
        createdAt: true,
        updatedAt: true,
        name: true,
        description: true,
        dueDate: true,
        index: true,
        isComplete: true,
        reminder: async () => rel((await import("./reminder.js")).reminder, "full", { omit: "reminderItems" }),
    },
};
