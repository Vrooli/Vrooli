import { FocusMode, FocusModeCreateInput, FocusModeUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { FocusModeFilterShape, shapeFocusModeFilter } from "./focusModeFilter";
import { LabelShape, shapeLabel } from "./label";
import { ReminderListShape, shapeReminderList } from "./reminderList";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { ScheduleShape, shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type FocusModeShape = Pick<FocusMode, "id" | "name" | "description"> & {
    __typename: "FocusMode",
    reminderList?: ReminderListShape | null,
    resourceList?: ResourceListShape | null;
    labels?: LabelShape[] | null,
    filters?: FocusModeFilterShape[] | null,
    schedule?: ScheduleShape | null,
}

export const shapeFocusMode: ShapeModel<FocusModeShape, FocusModeCreateInput, FocusModeUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "name", "description");
        return {
            ...prims,
            ...createRel(d, "reminderList", ["Create", "Connect"], "one", shapeReminderList),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "FocusMode" } })),
            ...createRel(d, "labels", ["Create", "Connect"], "many", shapeLabel),
            ...createRel(d, "filters", ["Create"], "many", shapeFocusModeFilter),
            ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "description"),
        ...updateRel(o, u, "reminderList", ["Create", "Connect", "Disconnect", "Update"], "one", shapeReminderList),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "FocusMode" } })),
        ...updateRel(o, u, "labels", ["Create", "Connect", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "filters", ["Create", "Delete"], "many", shapeFocusModeFilter),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }),
};
