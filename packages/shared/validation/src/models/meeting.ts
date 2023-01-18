import * as yup from 'yup';
import { description, eventEnd, eventStart, id, name, opt, recurrEnd, recurrStart, rel, req, timeZone, transRel, url, YupModel } from "../utils";
import { meetingInviteValidation } from './meetingInvite';

export const meetingTranslationValidation: YupModel = transRel({
    create: {
        name: opt(name),
        description: opt(description),
        link: opt(url),
    },
    update: {
        name: opt(name),
        description: opt(description),
        link: opt(url),
    },
})

export const meetingValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        openToAnyoneWithInvite: opt(yup.boolean()),
        showOnOrganizationProfile: opt(yup.boolean()),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(yup.boolean()),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
        ...rel('organization', ['Connect'], 'one', 'req'),
        ...rel('restrictedToRoles', ['Connect'], 'many', 'opt'),
        ...rel('invites', ['Create'], 'many', 'opt', meetingInviteValidation),
        ...rel('labels', ['Connect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', meetingTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        openToAnyoneWithInvite: opt(yup.boolean()),
        showOnOrganizationProfile: opt(yup.boolean()),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(yup.boolean()),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
        ...rel('restrictedToRoles', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('invites', ['Create', 'Update', 'Delete'], 'many', 'opt'),
        ...rel('labels', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', meetingTranslationValidation),
    }),

}