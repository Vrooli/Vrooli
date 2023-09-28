import * as yup from "yup";
import { blankToUndefined, description, id, maxStrErr, minNumErr, minStrErr, opt, req, YupModel, yupObj } from "../utils";

const index = yup.number().integer().min(0, minNumErr);

export const reminderItemValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(yup.string().trim().transform(blankToUndefined).min(1, minStrErr).max(50, maxStrErr)),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(yup.string().trim().transform(blankToUndefined).min(1, minStrErr).max(50, maxStrErr)),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [], [], o),
};
