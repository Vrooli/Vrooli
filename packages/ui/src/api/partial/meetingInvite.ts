import { MeetingInvite, MeetingInviteYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";
import { meetingPartial } from "./meeting";

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
        you: () => relPartial(require().meetingInviteYouPartial, 'full'),
    },
    full: {
        meeting: () => relPartial(require().meetingPartial, 'full', { omit: 'invites' }),

    },
    list: {
        meeting: () => relPartial(require().meetingPartial, 'list', { omit: 'invites' }),

    }
}