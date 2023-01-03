import { RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type RunRoutineShape = Pick<RunRoutine, 'id'>

export const shapeRunRoutine: ShapeModel<RunRoutineShape, RunRoutineCreateInput, RunRoutineUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}