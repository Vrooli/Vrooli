import { type Reminder } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reminder: ApiPartial<Reminder> = {
    full: {
        id: true,
        createdAt: true,
        updatedAt: true,
        name: true,
        description: true,
        dueDate: true,
        index: true,
        isComplete: true,
        reminderItems: async () => rel((await import("./reminderItem.js")).reminderItem, "full", { omit: "reminder" }),
        reminderList: async () => rel((await import("./reminderList.js")).reminderList, "full", { omit: "reminders" }),
    },
};
