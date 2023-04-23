import { id, req, yupObj } from "../utils";
import { reminderValidation } from "./reminder";
export const reminderListValidation = {
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
//# sourceMappingURL=reminderList.js.map