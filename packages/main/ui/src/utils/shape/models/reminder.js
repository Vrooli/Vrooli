import { shapeReminderItem } from "./reminderItem";
import { shapeReminderList } from "./reminderList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeReminder = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "description", "dueDate", "index"),
        ...createRel(d, "reminderList", ["Connect", "Create"], "one", shapeReminderList),
        ...createRel(d, "reminderItems", ["Create"], "many", shapeReminderItem, (r) => ({ ...r, reminder: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", "dueDate", "index"),
        ...updateRel(o, u, "reminderItems", ["Create", "Update", "Delete"], "many", shapeReminderItem, (r, i) => ({ ...r, reminder: { id: i.id } })),
    }, a),
};
//# sourceMappingURL=reminder.js.map