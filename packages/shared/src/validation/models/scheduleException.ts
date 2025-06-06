/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in scheduleException.test.ts
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, newEndTime, newStartTime, originalStartTime } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const scheduleExceptionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        originalStartTime: opt(originalStartTime),
        newStartTime: opt(newStartTime),
        newEndTime: req(newEndTime),
    }, [
        ["schedule", ["Connect"], "one", "req", ["exceptions"]],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        originalStartTime: opt(originalStartTime),
        newStartTime: opt(newStartTime),
        newEndTime: opt(newEndTime),
    }, [], [], d),
};
