import { MeetingInviteYou } from "@shared/consts";
import { GqlPartial } from "types";

export const meetingInviteYouPartial: GqlPartial<MeetingInviteYou> = {
    __typename: 'MeetingInviteYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const listMeetingInviteFields = ['MeetingInvite', `{
    id
}`] as const;
export const meetingInviteFields = ['MeetingInvite', `{
    id
}`] as const;