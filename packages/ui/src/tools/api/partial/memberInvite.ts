import { MemberInvite, MemberInviteYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const memberInviteYou: GqlPartial<MemberInviteYou> = {
    __typename: "MemberInviteYou",
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};

export const memberInvite: GqlPartial<MemberInvite> = {
    __typename: "MemberInvite",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        willBeAdmin: true,
        willHavePermissions: true,
        organization: async () => rel((await import("./organization")).organization, "nav"),
        user: async () => rel((await import("./user")).user, "nav"),
        you: () => rel(memberInviteYou, "full"),
    },
    full: {},
    list: {},
};
