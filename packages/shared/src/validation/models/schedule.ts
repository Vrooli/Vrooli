import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { endTime, id, startTime, timezone } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { scheduleExceptionValidation } from "./scheduleException.js";
import { scheduleRecurrenceValidation } from "./scheduleRecurrence.js";

export const scheduleValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        startTime: opt(startTime),
        endTime: opt(endTime),
        timezone: req(timezone),
    }, [
        ["exceptions", ["Create"], "many", "opt", scheduleExceptionValidation, ["schedule"]],
        ["meeting", ["Connect"], "one", "opt"],
        ["recurrences", ["Create"], "many", "opt", scheduleRecurrenceValidation, ["schedule"]],
        ["runProject", ["Connect"], "one", "opt"],
        ["runRoutine", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        startTime: opt(startTime),
        endTime: opt(endTime),
        timezone: opt(timezone),
    }, [
        ["exceptions", ["Create", "Update", "Delete"], "many", "opt", scheduleExceptionValidation, ["schedule"]],
        ["recurrences", ["Create", "Update", "Delete"], "many", "opt", scheduleRecurrenceValidation, ["schedule"]],
    ], [], d),
};
