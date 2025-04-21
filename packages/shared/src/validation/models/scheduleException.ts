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
