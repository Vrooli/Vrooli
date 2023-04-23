import { shapeNodeEnd } from "./nodeEnd";
import { shapeNodeRoutineList } from "./nodeRoutineList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeNodeTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name"), a),
};
export const shapeNode = {
    create: (d) => ({
        ...createPrims(d, "id", "columnIndex", "nodeType", "rowIndex"),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "end", ["Create"], "one", shapeNodeEnd),
        ...createRel(d, "routineList", ["Create"], "one", shapeNodeRoutineList),
        ...createRel(d, "translations", ["Create"], "many", shapeNodeTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "columnIndex", "nodeType", "rowIndex"),
        ...updateRel(o, u, "routineVersion", ["Connect"], "one"),
        ...updateRel(o, u, "end", ["Update"], "one", shapeNodeEnd),
        ...updateRel(o, u, "routineList", ["Update"], "one", shapeNodeRoutineList),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeTranslation),
    }, a),
};
//# sourceMappingURL=node.js.map