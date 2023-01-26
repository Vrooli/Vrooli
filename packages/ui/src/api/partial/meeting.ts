import { Meeting, MeetingTranslation, MeetingYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const meetingTranslationPartial: GqlPartial<MeetingTranslation> = {
    __typename: 'MeetingTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        link: true,
        name: true,
    },
    full: {},
    list: {},
}

export const meetingYouPartial: GqlPartial<MeetingYou> = {
    __typename: 'MeetingYou',
    common: {
        canDelete: true,
        canEdit: true,
        canInvite: true,
    },
    full: {},
    list: {},
}

export const meetingPartial: GqlPartial<Meeting> = {
    __typename: 'Meeting',
    common: {
        id: true,
        openToAnyoneWithInvite: true,
        showOnOrganizationProfile: true,
        timeZone: true,
        eventStart: true,
        eventEnd: true,
        recurring: true,
        recurrStart: true,
        recurrEnd: true,
        organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
        restrictedToRoles: () => relPartial(require('./role').rolePartial, 'full'),
        attendeesCount: true,
        invitesCount: true,
        you: () => relPartial(meetingYouPartial, 'full'),
    },
    full: {
        attendees: () => relPartial(require('./user').userPartial, 'nav'),
        invites: () => relPartial(require('./meetingInvite').meetingInvitePartial, 'list', { omit: 'meeting' }),
        labels: () => relPartial(require('./label').labelPartial, 'full'),
        translations: () => relPartial(meetingTranslationPartial, 'full'),
    },
    list: {
        labels: () => relPartial(require('./label').labelPartial, 'list'),
        translations: () => relPartial(meetingTranslationPartial, 'list'),
    }
}