import { description, id, minNumErr, name, opt, req, YupModel, yupObj } from '../utils';
import * as yup from 'yup';

const index = yup.number().integer().min(0, minNumErr)

export const reminderItemValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        description: req(description),
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
}