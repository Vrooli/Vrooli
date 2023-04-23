import { shapeFocusModeFilter } from "./focusModeFilter";
import { shapeLabel } from "./label";
import { shapeReminderList } from "./reminderList";
import { shapeResourceList } from "./resourceList";
import { shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeFocusMode = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "description"),
        ...createRel(d, "reminderList", ["Create", "Connect"], "one", shapeReminderList),
        ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList),
        ...createRel(d, "labels", ["Create", "Connect"], "many", shapeLabel),
        ...createRel(d, "filters", ["Create"], "many", shapeFocusModeFilter),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description"),
        ...updateRel(o, u, "reminderList", ["Create", "Connect", "Disconnect", "Update"], "one", shapeReminderList),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList),
        ...updateRel(o, u, "labels", ["Create", "Connect", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "filters", ["Create", "Delete"], "many", shapeFocusModeFilter),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }, a),
};
//# sourceMappingURL=focusMode.js.map