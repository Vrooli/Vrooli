import { bool, description, eventEnd, eventStart, id, name, opt, recurrEnd, recurrStart, rel, req, timeZone, transRel, url, YupModel, yupObj } from "../utils";
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
    create: ({ o }) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnOrganizationProfile: opt(bool),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['organization', ['Connect'], 'one', 'req'],
        ['restrictedToRoles', ['Connect'], 'many', 'opt'],
        ['invites', ['Create'], 'many', 'opt', meetingInviteValidation],
        ['labels', ['Connect'], 'many', 'opt'],
        ['translations', ['Create'], 'many', 'opt', meetingTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnOrganizationProfile: opt(bool),
        timeZone: opt(timeZone),
        eventStart: opt(eventStart),
        eventEnd: opt(eventEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['restrictedToRoles', ['Connect', 'Disconnect'], 'many', 'opt'],
        ['invites', ['Create', 'Update', 'Delete'], 'many', 'opt'],
        ['labels', ['Connect', 'Disconnect'], 'many', 'opt'],
        ['translations', ['Create'], 'many', 'opt', meetingTranslationValidation],
    ], [], o),
}