import { ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type ScheduleExceptionShape = Pick<ScheduleException, 'id'> & {
    __typename?: 'ScheduleException';
}

export const shapeScheduleException: ShapeModel<ScheduleExceptionShape, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}