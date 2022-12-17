import { id, rel, req, YupModel } from '../utils';
import * as yup from 'yup';
import { reminderValidation } from './reminder';

export const reminderListValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        ...rel('userSchedule', ['Connect'], 'one', 'opt'),
        ...rel('reminders', ['Create'], 'many', 'opt', reminderValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        ...rel('reminders', ['Create', 'Update', 'Delete'], 'many', 'opt', reminderValidation),
    })
}