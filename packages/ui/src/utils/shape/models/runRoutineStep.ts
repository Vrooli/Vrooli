import { RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { NodeShape } from "./node";
import { RoutineVersionShape } from "./routineVersion";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type RunRoutineStepShape = Pick<RunRoutineStep, "id" | "contextSwitches" | "name" | "order" | "status" | "step" | "timeElapsed"> & {
    __typename?: "RunRoutineStep";
    node?: { id: string } | NodeShape | null;
    subroutineVersion?: { id: string } | RoutineVersionShape | null;
}

export const shapeRunRoutineStep: ShapeModel<RunRoutineStepShape, RunRoutineStepCreateInput, RunRoutineStepUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "contextSwitches", "name", "order", "status", "step", "timeElapsed"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "subroutineVersion", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "contextSwitches", "status", "timeElapsed"),
    }, a),
};
