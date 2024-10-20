import { ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ScheduleShape } from "./schedule";
import { createPrims, createRel, shapeDate, shapeUpdate, updatePrims } from "./tools";

export type ScheduleExceptionShape = Pick<ScheduleException, "id" | "originalStartTime" | "newStartTime" | "newEndTime"> & {
    __typename: "ScheduleException";
    schedule: CanConnect<ScheduleShape>;
}

export const shapeScheduleException: ShapeModel<ScheduleExceptionShape, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", ["originalStartTime", shapeDate], ["newStartTime", shapeDate], ["newEndTime", shapeDate]),
        ...createRel(d, "schedule", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", ["originalStartTime", shapeDate], ["newStartTime", shapeDate], ["newEndTime", shapeDate]),
    }),
};
