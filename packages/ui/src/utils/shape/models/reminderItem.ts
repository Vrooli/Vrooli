import { ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type ReminderItemShape = Pick<ReminderItem, 'id' | 'name' | 'description' | 'dueDate' | 'index'>

export const shapeReminderItem: ShapeModel<ReminderItemShape, ReminderItemCreateInput, ReminderItemUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'name', 'description', 'dueDate', 'index'),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'name', 'description', 'dueDate', 'index'),
    })
}