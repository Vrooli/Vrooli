import { MeetingInvite, MeetingInviteYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const meetingInviteYouPartial: GqlPartial<MeetingInviteYou> = {
    __typename: 'MeetingInviteYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const meetingInvitePartial: GqlPartial<MeetingInvite> = {
    __typename: 'MeetingInvite',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        you: () => relPartial(meetingInviteYouPartial, 'full'),
    },
    full: {
        meeting: async () => relPartial((await import('./meeting')).meetingPartial, 'full', { omit: 'invites' }),

    },
    list: {
        meeting: async () => relPartial((await import('./meeting')).meetingPartial, 'list', { omit: 'invites' }),

    }
}