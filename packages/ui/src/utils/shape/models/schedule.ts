import { Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { FocusModeShape } from "./focusMode";
import { LabelShape, shapeLabel } from "./label";
import { MeetingShape } from "./meeting";
import { RunProjectShape } from "./runProject";
import { RunRoutineShape } from "./runRoutine";
import { ScheduleExceptionShape, shapeScheduleException } from "./scheduleException";
import { ScheduleRecurrenceShape, shapeScheduleRecurrence } from "./scheduleRecurrence";
import { createPrims, createRel, shapeDate, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ScheduleShape = Pick<Schedule, "id" | "startTime" | "endTime" | "timezone"> & {
    __typename: "Schedule";
    exceptions: ScheduleExceptionShape[];
    focusMode?: { id: string } | FocusModeShape | null;
    labels?: LabelShape[] | null;
    meeting?: { id: string } | MeetingShape | null;
    recurrences: ScheduleRecurrenceShape[];
    runProject?: { id: string } | RunProjectShape | null;
    runRoutine?: { id: string } | RunRoutineShape | null;
}

export const shapeSchedule: ShapeModel<ScheduleShape, ScheduleCreateInput, ScheduleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", ["startTime", shapeDate], ["endTime", shapeDate], "timezone"),
        ...createRel(d, "exceptions", ["Create"], "many", shapeScheduleException, (e) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: d.id },
        })),
        ...createRel(d, "focusMode", ["Connect"], "one"),
        ...createRel(d, "labels", ["Create", "Connect"], "many", shapeLabel),
        ...createRel(d, "meeting", ["Connect"], "one"),
        ...createRel(d, "recurrences", ["Create"], "many", shapeScheduleRecurrence, (e) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: d.id },
        })),
        ...createRel(d, "runProject", ["Connect"], "one"),
        ...createRel(d, "runRoutine", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", ["startTime", shapeDate], ["endTime", shapeDate], "timezone"),
        ...updateRel(o, u, "exceptions", ["Create", "Update", "Delete"], "many", shapeScheduleException, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: i.id },
        })),
        ...updateRel(o, u, "labels", ["Create", "Connect", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "recurrences", ["Create", "Update", "Delete"], "many", shapeScheduleRecurrence, (e, i) => ({
            ...e,
            schedule: { __typename: "Schedule" as const, id: i.id },
        })),
    }, a),
};
