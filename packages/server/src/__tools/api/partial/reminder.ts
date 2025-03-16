import { Reminder } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminder: ApiPartial<Reminder> = {
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        index: true,
        isComplete: true,
        reminderItems: async () => rel((await import("./reminderItem.js")).reminderItem, "full", { omit: "reminder" }),
        reminderList: async () => rel((await import("./reminderList.js")).reminderList, "full", { omit: "reminders" }),
    },
};
