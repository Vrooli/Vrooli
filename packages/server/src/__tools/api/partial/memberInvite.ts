import { MemberInvite, MemberInviteYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const memberInviteYou: GqlPartial<MemberInviteYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const memberInvite: GqlPartial<MemberInvite> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        willBeAdmin: true,
        willHavePermissions: true,
        team: async () => rel((await import("./team")).team, "nav"),
        user: async () => rel((await import("./user")).user, "nav"),
        you: () => rel(memberInviteYou, "full"),
    },
};
