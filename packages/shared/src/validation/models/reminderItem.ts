import * as yup from "yup";
import { bool, description, id, maxStrErr, minNumErr, minStrErr, opt, req, YupModel, yupObj } from "../utils";

const index = yup.number().integer().min(0, minNumErr);
const name = yup.string().trim().removeEmptyString().min(1, minStrErr).max(50, maxStrErr);

export const reminderItemValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
        isComplete: opt(bool),
        name: req(name),
    }, [
        ["reminder", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
        isComplete: opt(bool),
        name: opt(name),
    }, [], [], d),
};
