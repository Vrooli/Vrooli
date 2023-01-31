import { MemberInvite, MemberInviteYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const memberInviteYouPartial: GqlPartial<MemberInviteYou> = {
    __typename: 'MemberInviteYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}

export const memberInvitePartial: GqlPartial<MemberInvite> = {
    __typename: 'MemberInvite',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        willBeAdmin: true,
        willHavePermissions: true,
        organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
        user: () => relPartial(require('./user').userPartial, 'nav'),
        you: () => relPartial(memberInviteYouPartial, 'full'),
    },
    full: {},
    list: {},
}