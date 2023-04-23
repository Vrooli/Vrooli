import { shapeLabel } from "./label";
import { shapeScheduleException } from "./scheduleException";
import { shapeScheduleRecurrence } from "./scheduleRecurrence";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeSchedule = {
    create: (d) => ({
        ...createPrims(d, "id", "startTime", "endTime", "timezone"),
        ...createRel(d, "exceptions", ["Create"], "many", shapeScheduleException, (e) => ({
            ...e,
            schedule: { __typename: "Schedule", id: d.id },
        })),
        ...createRel(d, "labels", ["Create", "Connect"], "many", shapeLabel),
        ...createRel(d, "recurrences", ["Create"], "many", shapeScheduleRecurrence, (e) => ({
            ...e,
            schedule: { __typename: "Schedule", id: d.id },
        })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "startTime", "endTime", "timezone"),
        ...updateRel(o, u, "exceptions", ["Create", "Update", "Delete"], "many", shapeScheduleException, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule", id: i.id },
        })),
        ...updateRel(o, u, "labels", ["Create", "Connect", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "recurrences", ["Create", "Update", "Delete"], "many", shapeScheduleRecurrence, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule", id: i.id },
        })),
    }, a),
};
//# sourceMappingURL=schedule.js.map