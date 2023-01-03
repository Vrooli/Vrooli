import { ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ReminderItemShape = Pick<ReminderItem, 'id'>

export const shapeReminderItem: ShapeModel<ReminderItemShape, ReminderItemCreateInput, ReminderItemUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}