import { ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ScheduleShape, shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type ScheduleRecurrenceShape = Pick<ScheduleRecurrence, 'id' | 'recurrenceType' | 'interval' | 'dayOfMonth' | 'dayOfWeek' | 'month' | 'endDate'> & {
    __typename?: 'ScheduleRecurrence';
    schedule: ScheduleShape | { __typename?: 'Schedule', id: string };
}

export const shapeScheduleRecurrence: ShapeModel<ScheduleRecurrenceShape, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'recurrenceType', 'interval', 'dayOfMonth', 'dayOfWeek', 'month', 'endDate'),
        ...createRel(d, 'schedule', ['Connect', 'Create'], 'one', shapeSchedule),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'recurrenceType', 'interval', 'dayOfMonth', 'dayOfWeek', 'month', 'endDate'),
    }, a)
}