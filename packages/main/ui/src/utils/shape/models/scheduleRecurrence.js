import { shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";
export const shapeScheduleRecurrence = {
    create: (d) => ({
        ...createPrims(d, "id", "recurrenceType", "interval", "dayOfMonth", "dayOfWeek", "month", "endDate"),
        ...createRel(d, "schedule", ["Connect", "Create"], "one", shapeSchedule),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "recurrenceType", "interval", "dayOfMonth", "dayOfWeek", "month", "endDate"),
    }, a),
};
//# sourceMappingURL=scheduleRecurrence.js.map