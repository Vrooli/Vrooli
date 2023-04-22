import { id, req, YupModel, yupObj } from "../utils";
import { reminderValidation } from "./reminder";

export const reminderListValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["focusMode", ["Connect"], "one", "opt"],
        ["reminders", ["Create"], "many", "opt", reminderValidation, ["reminderList"]],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["focusMode", ["Connect"], "one", "opt"],
        ["reminders", ["Create", "Update", "Delete"], "many", "opt", reminderValidation, ["reminderList"]],
    ], [], o),
};
