import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";
export const shapeScheduleException = {
    create: (d) => ({
        ...createPrims(d, "id", "originalStartTime", "newStartTime", "newEndTime"),
        ...createRel(d, "schedule", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "originalStartTime", "newStartTime", "newEndTime"),
    }, a),
};
//# sourceMappingURL=scheduleException.js.map