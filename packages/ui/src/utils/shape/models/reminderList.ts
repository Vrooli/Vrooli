import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type ReminderListShape = Pick<ReminderList, 'id'>

export const shapeReminderList: ShapeModel<ReminderListShape, ReminderListCreateInput, ReminderListUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}