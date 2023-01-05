import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ReminderShape, shapeReminder } from "./reminder";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ReminderListShape = Pick<ReminderList, 'id'> & {
    userSchedule?: { id: string },
    reminders?: ReminderShape[],
}

export const shapeReminderList: ShapeModel<ReminderListShape, ReminderListCreateInput, ReminderListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id'),
        ...createRel(d, 'userSchedule', ['Connect'], 'one'),
        ...createRel(d, 'reminders', ['Create'], 'many', shapeReminder, (r) => ({ ...r, reminderList: { id: d.id } })),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateRel(o, u, 'reminders', ['Create', 'Update', 'Delete'], 'many', shapeReminder, (r, i) => ({ ...r, reminderList: { id: i.id } })),
    })
}