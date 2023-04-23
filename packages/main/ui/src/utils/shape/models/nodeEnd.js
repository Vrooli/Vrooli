import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeNodeEnd = {
    create: (d) => ({
        ...createPrims(d, "id", "wasSuccessful"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "suggestedNextRoutineVersions", ["Connect"], "many"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "wasSuccessful"),
        ...updateRel(o, u, "suggestedNextRoutineVersions", ["Connect", "Disconnect"], "many"),
    }, a),
};
//# sourceMappingURL=nodeEnd.js.map