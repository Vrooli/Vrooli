import { UserScheduleFilter, UserScheduleFilterCreateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type UserScheduleFilterShape = Pick<UserScheduleFilter, 'id'>

export const shapeUserScheduleFilter: ShapeModel<UserScheduleFilterShape, UserScheduleFilterCreateInput, null> = {
    create: (item) => ({}) as any
}