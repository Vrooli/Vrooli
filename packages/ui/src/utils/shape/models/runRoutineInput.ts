import { RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type RunRoutineInputShape = Pick<RunRoutineInput, 'id' | 'data'> & {
    input: { id: string };
    runRoutine: { id: string };
}

export const shapeRunRoutineInput: ShapeModel<RunRoutineInputShape, RunRoutineInputCreateInput, RunRoutineInputUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'data'),
        ...createRel(item, 'input', ['Connect'], 'one'),
        ...createRel(item, 'runRoutine', ['Connect'], 'one'),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'data')
    })
}