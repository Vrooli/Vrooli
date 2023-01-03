import { RunRoutineSchedule, RunRoutineScheduleCreateInput, RunRoutineScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type RunRoutineScheduleShape = Pick<RunRoutineSchedule, 'id'>

export const shapeRunRoutineSchedule: ShapeModel<RunRoutineScheduleShape, RunRoutineScheduleCreateInput, RunRoutineScheduleUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}