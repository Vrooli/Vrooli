import { endTime, id, opt, req, startTime, timezone, YupModel, yupObj } from "../utils";
import { focusModeValidation } from "./focusMode";
import { labelValidation } from "./label";
import { meetingValidation } from "./meeting";
import { runProjectValidation } from "./runProject";
import { runRoutineValidation } from "./runRoutine";
import { scheduleExceptionValidation } from "./scheduleException";
import { scheduleRecurrenceValidation } from "./scheduleRecurrence";

export const scheduleValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        startTime: opt(startTime),
        endTime: opt(endTime),
        timezone: req(timezone),
    }, [
        ["exceptions", ["Create"], "many", "opt", scheduleExceptionValidation],
        ["focusMode", ["Connect"], "one", "opt", focusModeValidation],
        ["labels", ["Create", "Connect"], "many", "opt", labelValidation],
        ["meeting", ["Connect"], "one", "opt", meetingValidation],
        ["recurrences", ["Create"], "many", "opt", scheduleRecurrenceValidation],
        ["runProject", ["Connect"], "one", "opt", runProjectValidation],
        ["runRoutine", ["Connect"], "one", "opt", runRoutineValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        startTime: opt(startTime),
        endTime: opt(endTime),
        timezone: opt(timezone),
    }, [
        ["exceptions", ["Create", "Update", "Delete"], "many", "opt", scheduleExceptionValidation],
        ["labels", ["Create", "Connect", "Disconnect"], "many", "opt", labelValidation],
        ["recurrences", ["Create", "Update", "Delete"], "many", "opt", scheduleRecurrenceValidation],
    ], [], d),
};
