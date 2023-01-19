import { MeetingYou } from "@shared/consts";
import { GqlPartial } from "types";

export const meetingYouPartial: GqlPartial<MeetingYou> = {
    __typename: 'MeetingYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canInvite: true,
    }),
}

export const listMeetingFields = ['Meeting', `{
    id
}`] as const;
export const meetingFields = ['Meeting', `{
    id
}`] as const;