import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type RunProjectStepShape = Pick<RunProjectStep, 'id'>

export const shapeRunProjectStep: ShapeModel<RunProjectStepShape, RunProjectStepCreateInput, RunProjectStepUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}