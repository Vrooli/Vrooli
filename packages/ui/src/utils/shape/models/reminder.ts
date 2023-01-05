import { Reminder, ReminderCreateInput, ReminderUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ReminderItemShape, shapeReminderItem } from "./reminderItem";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ReminderShape = Pick<Reminder, 'id' | 'name' | 'description' | 'dueDate' | 'index'> & {
    reminderList: { id: string },
    reminderItems?: ReminderItemShape[],
}

export const shapeReminder: ShapeModel<ReminderShape, ReminderCreateInput, ReminderUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'name', 'description', 'dueDate', 'index'),
        ...createRel(d, 'reminderList', ['Connect'], 'one'),
        ...createRel(d, 'reminderItems', ['Create'], 'many', shapeReminderItem, (r) => ({ ...r, reminder: { id: d.id } })),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'name', 'description', 'dueDate', 'index'),
        ...updateRel(o, u, 'reminderItems', ['Create', 'Update', 'Delete'], 'many', shapeReminderItem, (r, i) => ({ ...r, reminder: { id: i.id }})),
    })
}