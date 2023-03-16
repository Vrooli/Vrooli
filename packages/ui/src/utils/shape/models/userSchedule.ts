import { UserSchedule, UserScheduleCreateInput, UserScheduleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { ReminderListShape, shapeReminderList } from "./reminderList";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
import { shapeUserScheduleFilter, UserScheduleFilterShape } from "./userScheduleFilter";

export type UserScheduleShape = Pick<UserSchedule, 'id' | 'name' | 'description' | 'timeZone' | 'eventStart' | 'eventEnd' | 'recurring' | 'recurrStart' | 'recurrEnd'> & {
    __typename?: 'UserSchedule',
    reminderList?: ReminderListShape | null,
    resourceList?: ResourceListShape | null,
    labels?: LabelShape[] | null,
    filters?: UserScheduleFilterShape[] | null,
}

export const shapeUserSchedule: ShapeModel<UserScheduleShape, UserScheduleCreateInput, UserScheduleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d,
            'id',
            'name',
            'description',
            'timeZone',
            'eventStart',
            'eventEnd',
            'recurring',
            'recurrStart',
            'recurrEnd'),
        ...createRel(d, 'reminderList', ['Create', 'Connect'], 'one', shapeReminderList),
        ...createRel(d, 'resourceList', ['Create'], 'one', shapeResourceList),
        ...createRel(d, 'labels', ['Create', 'Connect'], 'many', shapeLabel),
        ...createRel(d, 'filters', ['Create'], 'many', shapeUserScheduleFilter),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u,
            'id',
            'name',
            'description',
            'timeZone',
            'eventStart',
            'eventEnd',
            'recurring',
            'recurrStart',
            'recurrEnd'),
        ...updateRel(o, u, 'reminderList', ['Create', 'Connect', 'Disconnect', 'Update'], 'one', shapeReminderList),
        ...updateRel(o, u, 'resourceList', ['Create', 'Update'], 'one', shapeResourceList),
        ...updateRel(o, u, 'labels', ['Create', 'Connect', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'filters', ['Create', 'Delete'], 'many', shapeUserScheduleFilter),
    }, a)
}