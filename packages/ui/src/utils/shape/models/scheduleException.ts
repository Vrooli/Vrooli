import { ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ScheduleShape } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type ScheduleExceptionShape = Pick<ScheduleException, "id" | "originalStartTime" | "newStartTime" | "newEndTime"> & {
    __typename?: "ScheduleException";
    schedule: ScheduleShape | { __typename?: "Schedule", id: string };
}

export const shapeScheduleException: ShapeModel<ScheduleExceptionShape, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "originalStartTime", "newStartTime", "newEndTime"),
        ...createRel(d, "schedule", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "originalStartTime", "newStartTime", "newEndTime"),
    }, a),
};
