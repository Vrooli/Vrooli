import { ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ReminderShape } from "./reminder";
import { createPrims, createRel, shapeDate, shapeUpdate, updatePrims } from "./tools";

export type ReminderItemShape = Pick<ReminderItem, "id" | "name" | "description" | "dueDate" | "index" | "isComplete"> & {
    __typename: "ReminderItem";
    reminder: CanConnect<ReminderShape>;
}

export const shapeReminderItem: ShapeModel<ReminderItemShape, ReminderItemCreateInput, ReminderItemUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
        ...createRel(d, "reminder", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description", ["dueDate", shapeDate], "index", "isComplete"),
    }, a),
};
