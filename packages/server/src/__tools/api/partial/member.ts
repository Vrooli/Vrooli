import { Member, MemberYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const memberYou: ApiPartial<MemberYou> = {
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const member: ApiPartial<Member> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isAdmin: true,
        permissions: true,
        roles: async () => rel((await import("./role.js")).role, "list"),
        you: () => rel(memberYou, "full"),
    },
    full: {
        team: async () => rel((await import("./team.js")).team, "full"),
        user: async () => rel((await import("./user.js")).user, "full"),
    },
    list: {
        team: async () => rel((await import("./team.js")).team, "list"),
        user: async () => rel((await import("./user.js")).user, "list"),
    },
};
