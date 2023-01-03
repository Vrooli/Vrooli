import { UserSchedule, UserScheduleCreateInput, UserScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type UserScheduleShape = Pick<UserSchedule, 'id'>

export const shapeUserSchedule: ShapeModel<UserScheduleShape, UserScheduleCreateInput, UserScheduleUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}