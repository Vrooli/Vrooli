import { RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type RunRoutineStepShape = Pick<RunRoutineStep, 'id'>

export const shapeRunRoutineStep: ShapeModel<RunRoutineStepShape, RunRoutineStepCreateInput, RunRoutineStepUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}