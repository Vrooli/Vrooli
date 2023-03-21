import { Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type ScheduleShape = Pick<Schedule, 'id'> & {
    __typename?: 'Schedule';
}

export const shapeSchedule: ShapeModel<ScheduleShape, ScheduleCreateInput, ScheduleUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}