import { Meeting, MeetingTranslation, MeetingYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const meetingTranslation: GqlPartial<MeetingTranslation> = {
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

export const meetingYou: GqlPartial<MeetingYou> = {
    __typename: 'MeetingYou',
    common: {
        canDelete: true,
        canEdit: true,
        canInvite: true,
    },
    full: {},
    list: {},
}

export const meeting: GqlPartial<Meeting> = {
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
        organization: async () => rel((await import('./organization')).organization, 'nav'),
        restrictedToRoles: async () => rel((await import('./role')).role, 'full'),
        attendeesCount: true,
        invitesCount: true,
        you: () => rel(meetingYou, 'full'),
    },
    full: {
        __define: {
            0: async () => rel((await import('./label')).label, 'full'),
        },
        attendees: async () => rel((await import('./user')).user, 'nav'),
        invites: async () => rel((await import('./meetingInvite')).meetingInvite, 'list', { omit: 'meeting' }),
        labels: { __use: 0 },
        translations: () => rel(meetingTranslation, 'full'),
    },
    list: {
        __define: {
            0: async () => rel((await import('./label')).label, 'list'),
        },
        labels: { __use: 0 },
        translations: () => rel(meetingTranslation, 'list'),
    }
}