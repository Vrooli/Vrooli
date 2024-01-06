import * as yup from "yup";
import { description, id, index, name, opt, req, YupModel, yupObj } from "../utils";
import { reminderItemValidation } from "./reminderItem";
import { reminderListValidation } from "./reminderList";

export const reminderValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        name: req(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ["reminderList", ["Connect", "Create"], "one", "req", reminderListValidation, ["reminders"]],
        ["reminderItems", ["Create"], "many", "opt", reminderItemValidation],
    ], [["reminderListConnect", "reminderListCreate"]], d),
    update: (d) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ["reminderList", ["Connect", "Create"], "one", "opt", reminderListValidation, ["reminders"]],
        ["reminderItems", ["Create", "Update", "Delete"], "many", "opt", reminderItemValidation],
    ], [], d),
};
