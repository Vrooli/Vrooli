/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in schedule.test.ts
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { endTime, id, startTime, timezone } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { scheduleExceptionValidation } from "./scheduleException.js";
import { scheduleRecurrenceValidation } from "./scheduleRecurrence.js";

export const scheduleValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        startTime: req(startTime),
        endTime: req(endTime),
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
