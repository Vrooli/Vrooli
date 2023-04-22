import { ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput } from "@shared/consts";
import { ShapeModel } from "../../../types";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type ReminderItemShape = Pick<ReminderItem, "id" | "name" | "description" | "dueDate" | "index"> & {
    __typename?: "ReminderItem";
    reminder: { id: string },
}

export const shapeReminderItem: ShapeModel<ReminderItemShape, ReminderItemCreateInput, ReminderItemUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "description", "dueDate", "index"),
        ...createRel(d, "reminder", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", "dueDate", "index"),
    }, a),
};
