import { rel } from "../utils";
export const reminderItem = {
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
//# sourceMappingURL=reminderItem.js.map