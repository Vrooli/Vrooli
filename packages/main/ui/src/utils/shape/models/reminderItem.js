import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";
export const shapeReminderItem = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "description", "dueDate", "index"),
        ...createRel(d, "reminder", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", "dueDate", "index"),
    }, a),
};
//# sourceMappingURL=reminderItem.js.map