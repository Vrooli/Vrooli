import { ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ScheduleShape, shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeDate, shapeUpdate, updatePrims } from "./tools";

export type ScheduleRecurrenceShape = Pick<ScheduleRecurrence, "id" | "recurrenceType" | "interval" | "dayOfMonth" | "dayOfWeek" | "duration" | "month" | "endDate"> & {
    __typename: "ScheduleRecurrence";
    schedule: CanConnect<ScheduleShape>;
}

export const shapeScheduleRecurrence: ShapeModel<ScheduleRecurrenceShape, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "recurrenceType", "interval", "dayOfMonth", "dayOfWeek", "duration", "month", ["endDate", shapeDate]),
        ...createRel(d, "schedule", ["Connect", "Create"], "one", shapeSchedule),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "recurrenceType", "interval", "dayOfMonth", "dayOfWeek", "duration", "month", ["endDate", shapeDate]),
    }),
};
