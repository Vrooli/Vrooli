import { MemberInvite, MemberInviteYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { organizationPartial } from "./organization";
import { userPartial } from "./user";

export const memberInviteYouPartial: GqlPartial<MemberInviteYou> = {
    __typename: 'MemberInviteYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
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
        organization: () => relPartial(organizationPartial, 'nav'),
        user: () => relPartial(userPartial, 'nav'),
        you: () => relPartial(memberInviteYouPartial, 'full'),
    },
}