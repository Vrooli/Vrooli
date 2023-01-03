import { Reminder, ReminderCreateInput, ReminderUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ReminderShape = Pick<Reminder, 'id'>

export const shapeReminder: ShapeModel<ReminderShape, ReminderCreateInput, ReminderUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}