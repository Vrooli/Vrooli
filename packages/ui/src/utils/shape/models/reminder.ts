import { Reminder, ReminderCreateInput, ReminderUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ReminderItemShape, shapeReminderItem } from "./reminderItem";
import { ReminderListShape, shapeReminderList } from "./reminderList";
import { createPrims, createRel, shapeDate, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ReminderShape = Pick<Reminder, "id" | "name" | "description" | "dueDate" | "index" | "isComplete"> & {
    __typename: "Reminder";
    reminderList: CanConnect<ReminderListShape>;
    reminderItems?: CanConnect<ReminderItemShape>[] | null;
}

export const shapeReminder: ShapeModel<ReminderShape, ReminderCreateInput, ReminderUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "name", "description", ["dueDate", shapeDate], "index");
        return {
            ...prims,
            ...createRel(d, "reminderItems", ["Create"], "many", shapeReminderItem, (r) => ({ ...r, reminder: { id: prims.id } })),
            // Treat as connect when the reminderList has created_at
            ...createRel(d, "reminderList", ["Connect", "Create"], "one", shapeReminderList, (l) => {
                if (l.created_at) return { id: l.id };
                return l;
            }),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
        ...updateRel(o, u, "reminderItems", ["Create", "Update", "Delete"], "many", shapeReminderItem, (r, i) => ({ ...r, reminder: { id: i.id } })),
        // Treat as connect when the reminderList has created_at
        ...updateRel(o, u, "reminderList", ["Connect", "Create", "Disconnect"], "one", shapeReminderList, (l) => {
            if (l.created_at) return { id: l.id };
            return l;
        }),
    }, a),
};
