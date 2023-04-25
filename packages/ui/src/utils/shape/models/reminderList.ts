import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { FocusModeShape } from "./focusMode";
import { ReminderShape, shapeReminder } from "./reminder";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ReminderListShape = Pick<ReminderList, 'id'> & {
    __typename?: 'ReminderList';
    focusMode?: { id: string } | FocusModeShape | null,
    reminders?: ReminderShape[] | null,
}

export const shapeReminderList: ShapeModel<ReminderListShape, ReminderListCreateInput, ReminderListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id'),
        ...createRel(d, 'focusMode', ['Connect'], 'one'),
        ...createRel(d, 'reminders', ['Create'], 'many', shapeReminder, (r) => ({ ...r, reminderList: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateRel(o, u, 'reminders', ['Create', 'Update', 'Delete'], 'many', shapeReminder, (r, i) => ({ ...r, reminderList: { id: i.id } })),
    }, a)
}