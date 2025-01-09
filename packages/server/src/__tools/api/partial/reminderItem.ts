import { ReminderItem } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        reminder: async () => rel((await import("./reminder")).reminder, "full", { omit: "reminderItems" }),
    },
};
