import { RoutineVersionInput, RunRoutine, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type RunRoutineInputShape = Pick<RunRoutineInput, "id" | "data"> & {
    __typename: "RunRoutineInput";
    input: CanConnect<RoutineVersionInput>;
    runRoutine: CanConnect<RunRoutine>;
}

export const shapeRunRoutineInput: ShapeModel<RunRoutineInputShape, RunRoutineInputCreateInput, RunRoutineInputUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "data"),
        ...createRel(d, "input", ["Connect"], "one"),
        ...createRel(d, "runRoutine", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "data"),
    }, a),
};
