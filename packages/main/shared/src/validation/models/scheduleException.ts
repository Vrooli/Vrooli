import { id, newEndTime, newStartTime, opt, originalStartTime, req, YupModel, yupObj } from "../utils";
import { scheduleValidation } from "./schedule";

export const scheduleExceptionValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        originalStartTime: opt(originalStartTime),
        newStartTime: opt(newStartTime),
        newEndTime: req(newEndTime),
    }, [
        ["schedule", ["Connect"], "one", "req", scheduleValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        originalStartTime: opt(originalStartTime),
        newStartTime: opt(newStartTime),
        newEndTime: opt(newEndTime),
    }, [], [], o),
};
