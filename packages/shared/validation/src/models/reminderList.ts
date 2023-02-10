import { id, req, YupModel, yupObj } from '../utils';
import { reminderValidation } from './reminder';

export const reminderListValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['userSchedule', ['Connect'], 'one', 'opt'],
        ['reminders', ['Create'], 'many', 'opt', reminderValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ['reminders', ['Create', 'Update', 'Delete'], 'many', 'opt', reminderValidation],
    ], [], o),
}