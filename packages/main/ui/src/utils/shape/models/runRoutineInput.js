import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";
export const shapeRunRoutineInput = {
    create: (d) => ({
        ...createPrims(d, "id", "data"),
        ...createRel(d, "input", ["Connect"], "one"),
        ...createRel(d, "runRoutine", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "data"),
    }, a),
};
//# sourceMappingURL=runRoutineInput.js.map