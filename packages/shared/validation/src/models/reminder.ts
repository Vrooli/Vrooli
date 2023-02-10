import { description, id, index, name, opt, req, YupModel, yupObj } from '../utils';
import * as yup from 'yup';
import { reminderItemValidation } from './reminderItem';

export const reminderValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        description: req(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ['reminderList', ['Connect'], 'one', 'req'],
        ['reminderItems', ['Create'], 'many', 'opt', reminderItemValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
        dueDate: opt(yup.date()),
        index: opt(index),
    }, [
        ['reminderItems', ['Create', 'Update', 'Delete'], 'many', 'opt', reminderItemValidation],
    ], [], o),
}