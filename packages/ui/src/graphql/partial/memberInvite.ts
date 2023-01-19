import { MemberInviteYou } from "@shared/consts";
import { GqlPartial } from "types";

export const memberInviteYouPartial: GqlPartial<MemberInviteYou> = {
    __typename: 'MemberInviteYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
    }),
}

export const listMemberInviteFields = ['MemberInvite', `{
    id
}`] as const;
export const memberInviteFields = ['MemberInvite', `{
    id
}`] as const;