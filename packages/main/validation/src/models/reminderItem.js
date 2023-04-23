import * as yup from "yup";
import { description, id, minNumErr, name, opt, req, yupObj } from "../utils";
const index = yup.number().integer().min(0, minNumErr);
export const reminderItemValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [], [], o),
};
//# sourceMappingURL=reminderItem.js.map