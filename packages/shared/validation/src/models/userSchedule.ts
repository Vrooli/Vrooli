import { bool, description, eventEnd, eventStart, id, name, opt, recurrEnd, recurrStart, req, timeZone, YupModel, yupObj } from "../utils";
import { labelValidation } from './label';
import { reminderListValidation } from './reminderList';
import { resourceListValidation } from './resourceList';
import { userScheduleFilterValidation } from './userScheduleFilter';

export const userScheduleValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        name: req(name),
        description: opt(description),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['reminderList', ['Create', 'Connect'], 'one', 'opt', reminderListValidation],
        ['resourceList', ['Create'], 'one', 'opt', resourceListValidation],
        ['labels', ['Create', 'Connect'], 'many', 'opt', labelValidation],
        ['filters', ['Create'], 'many', 'opt', userScheduleFilterValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['reminderList', ['Connect', 'Disconnect', 'Create', 'Update'], 'one', 'opt', reminderListValidation],
        ['resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation],
        ['labels', ['Create', 'Connect', 'Disconnect'], 'many', 'opt', labelValidation],
        ['filters', ['Create', 'Delete'], 'many', 'opt', userScheduleFilterValidation],
    ], [], o),
}