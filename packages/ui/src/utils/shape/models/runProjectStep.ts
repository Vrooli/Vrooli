import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunProjectStepShape = Pick<RunProjectStep, 'id'>

export const shapeRunProjectStep: ShapeModel<RunProjectStepShape, RunProjectStepCreateInput, RunProjectStepUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}