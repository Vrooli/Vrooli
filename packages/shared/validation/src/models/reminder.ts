import { description, id, index, name, opt, rel, req, YupModel } from '../utils';
import * as yup from 'yup';
import { reminderItemValidation } from './reminderItem';

export const reminderValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        name: req(name),
        description: req(description),
        dueDate: opt(yup.date()),
        index: opt(index),
        ...rel('reminderList', ['Connect'], 'one', 'req'),
        ...rel('reminderItems', ['Create'], 'many', 'opt', reminderItemValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        name: opt(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
        ...rel('reminderItems', ['Create', 'Update', 'Delete'], 'many', 'opt', reminderItemValidation),
    }),
}