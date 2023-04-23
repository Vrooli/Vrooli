import { shapeNodeRoutineListItem } from "./nodeRoutineListItem";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeNodeRoutineList = {
    create: (d) => ({
        ...createPrims(d, "id", "isOptional", "isOrdered"),
        ...createRel(d, "items", ["Create"], "many", shapeNodeRoutineListItem),
        ...createRel(d, "node", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isOptional", "isOrdered"),
        ...updateRel(o, u, "items", ["Create", "Update", "Delete"], "many", shapeNodeRoutineListItem),
    }, a),
};
//# sourceMappingURL=nodeRoutineList.js.map