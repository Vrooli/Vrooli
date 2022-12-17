import * as yup from 'yup';
import { description, eventEnd, eventStart, id, name, opt, recurrEnd, recurrStart, rel, req, timeZone, YupModel } from "../utils";
import { labelValidation } from './label';
import { reminderListValidation } from './reminderList';
import { resourceListValidation } from './resourceList';
import { userScheduleFilterValidation } from './userScheduleFilter';

export const userScheduleValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        name: req(name),
        description: opt(description),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(yup.boolean()),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
        ...rel('reminderList', ['Create', 'Connect'], 'one', 'opt', reminderListValidation),
        ...rel('resourceList', ['Create'], 'one', 'opt', resourceListValidation),
        ...rel('labels', ['Create', 'Connect'], 'many', 'opt', labelValidation),
        ...rel('filters', ['Create'], 'many', 'opt', userScheduleFilterValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        name: opt(name),
        description: opt(description),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(yup.boolean()),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
        ...rel('reminderList', ['Connect', 'Disconnect', 'Create', 'Update'], 'one', 'opt', reminderListValidation),
        ...rel('resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation),
        ...rel('labels', ['Create', 'Connect', 'Disconnect'], 'many', 'opt', labelValidation),
        ...rel('filters', ['Create', 'Delete'], 'many', 'opt', userScheduleFilterValidation),
    }),
}