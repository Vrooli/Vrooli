import { RoutineVersionInput, RunRoutine, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
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
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "data"),
    }),
};
