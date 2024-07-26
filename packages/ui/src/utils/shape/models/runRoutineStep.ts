import { RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { NodeShape } from "./node";
import { RoutineVersionShape } from "./routineVersion";
import { RunRoutineShape } from "./runRoutine";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type RunRoutineStepShape = Pick<RunRoutineStep, "id" | "contextSwitches" | "name" | "order" | "status" | "step" | "timeElapsed"> & {
    __typename: "RunRoutineStep";
    node?: CanConnect<NodeShape> | null;
    runRoutine: CanConnect<RunRoutineShape>;
    subroutine?: CanConnect<RoutineVersionShape> | null;
}

export const shapeRunRoutineStep: ShapeModel<RunRoutineStepShape, RunRoutineStepCreateInput, RunRoutineStepUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "contextSwitches", "name", "order", "status", "step", "timeElapsed"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "runRoutine", ["Connect"], "one"),
        ...createRel(d, "subroutine", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "contextSwitches", "status", "timeElapsed"),
    }, a),
};
