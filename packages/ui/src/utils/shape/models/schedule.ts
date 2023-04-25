import { Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { ScheduleExceptionShape, shapeScheduleException } from "./scheduleException";
import { ScheduleRecurrenceShape, shapeScheduleRecurrence } from "./scheduleRecurrence";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ScheduleShape = Pick<Schedule, 'id' | 'startTime' | 'endTime' | 'timezone'> & {
    __typename?: 'Schedule';
    exceptions: ScheduleExceptionShape[];
    labels?: LabelShape[] | null;
    recurrences: ScheduleRecurrenceShape[];
}

export const shapeSchedule: ShapeModel<ScheduleShape, ScheduleCreateInput, ScheduleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'startTime', 'endTime', 'timezone'),
        ...createRel(d, 'exceptions', ['Create'], 'many', shapeScheduleException, (e) => ({
            ...e,
            schedule: { __typename: 'Schedule' as const, id: d.id }
        })),
        ...createRel(d, 'labels', ['Create', 'Connect'], 'many', shapeLabel),
        ...createRel(d, 'recurrences', ['Create'], 'many', shapeScheduleRecurrence, (e) => ({
            ...e,
            schedule: { __typename: 'Schedule' as const, id: d.id }
        })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'startTime', 'endTime', 'timezone'),
        ...updateRel(o, u, 'exceptions', ['Create', 'Update', 'Delete'], 'many', shapeScheduleException, (e, i) => ({
            ...e,
            schedule: { __typename: 'Schedule' as const, id: i.id }
        })),
        ...updateRel(o, u, 'labels', ['Create', 'Connect', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'recurrences', ['Create', 'Update', 'Delete'], 'many', shapeScheduleRecurrence, (e, i) => ({
            ...e,
            schedule: { __typename: 'Schedule' as const, id: i.id }
        })),
    }, a)
}