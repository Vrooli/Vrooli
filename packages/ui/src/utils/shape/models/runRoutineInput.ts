import { RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type RunRoutineInputShape = Pick<RunRoutineInput, 'id' | 'data'> & {
    input: { id: string };
    runRoutine: { id: string };
}

export const shapeRunRoutineInput: ShapeModel<RunRoutineInputShape, RunRoutineInputCreateInput, RunRoutineInputUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'data'),
        ...createRel(d, 'input', ['Connect'], 'one'),
        ...createRel(d, 'runRoutine', ['Connect'], 'one'),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'data')
    }, a)
}