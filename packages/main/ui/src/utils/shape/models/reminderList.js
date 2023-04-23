import { shapeReminder } from "./reminder";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeReminderList = {
    create: (d) => ({
        ...createPrims(d, "id"),
        ...createRel(d, "focusMode", ["Connect"], "one"),
        ...createRel(d, "reminders", ["Create"], "many", shapeReminder, (r) => ({ ...r, reminderList: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "reminders", ["Create", "Update", "Delete"], "many", shapeReminder, (r, i) => ({ ...r, reminderList: { id: i.id } })),
    }, a),
};
//# sourceMappingURL=reminderList.js.map