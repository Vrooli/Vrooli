import { Reminder } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reminder: GqlPartial<Reminder> = {
    __typename: "Reminder",
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        index: true,
        isComplete: true,
        reminderItems: async () => rel((await import("./reminderItem")).reminderItem, "full", { omit: "reminder" }),
        reminderList: async () => rel((await import("./reminderList")).reminderList, "nav", { omit: "reminders" }),
    },
};
