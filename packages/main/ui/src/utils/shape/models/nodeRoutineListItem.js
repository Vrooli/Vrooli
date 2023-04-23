import { shapeRoutineVersion } from "./routineVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeNodeRoutineListItemTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name"), a),
};
export const shapeNodeRoutineListItem = {
    create: (d) => ({
        ...createPrims(d, "id", "index", "isOptional"),
        ...createRel(d, "list", ["Connect"], "one"),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeNodeRoutineListItemTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "index", "isOptional"),
        ...updateRel(o, u, "routineVersion", ["Update"], "one", shapeRoutineVersion),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeRoutineListItemTranslation),
    }, a),
};
//# sourceMappingURL=nodeRoutineListItem.js.map