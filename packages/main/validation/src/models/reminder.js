import * as yup from "yup";
import { description, id, index, name, opt, req, yupObj } from "../utils";
import { reminderItemValidation } from "./reminderItem";
import { reminderListValidation } from "./reminderList";
export const reminderValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ["reminderList", ["Connect", "Create"], "one", "req", reminderListValidation, ["reminders"]],
        ["reminderItems", ["Create"], "many", "opt", reminderItemValidation],
    ], [["reminderListConnect", "reminderListCreate"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ["reminderItems", ["Create", "Update", "Delete"], "many", "opt", reminderItemValidation],
    ], [], o),
};
//# sourceMappingURL=reminder.js.map