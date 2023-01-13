import { RunProject, RunProjectCreateInput, RunProjectUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type RunProjectShape = Pick<RunProject, 'id'>

export const shapeRunProject: ShapeModel<RunProjectShape, RunProjectCreateInput, RunProjectUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}