import { MemberInvite, MemberInviteYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const memberInviteYou: ApiPartial<MemberInviteYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const memberInvite: ApiPartial<MemberInvite> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        message: true,
        status: true,
        willBeAdmin: true,
        willHavePermissions: true,
        team: async () => rel((await import("./team.js")).team, "nav"),
        user: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(memberInviteYou, "full"),
    },
};
