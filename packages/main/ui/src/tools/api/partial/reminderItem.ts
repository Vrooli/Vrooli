import { ReminderItem } from ":local/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reminderItem: GqlPartial<ReminderItem> = {
    __typename: "ReminderItem",
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        index: true,
        isComplete: true,
        reminder: async () => rel((await import("./reminder")).reminder, "nav", { omit: "reminderItems" }),
    },
};
