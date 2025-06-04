import { type Member, type MemberYou } from "@local/shared";
import { type ApiPartial } from "../types.js";
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
        publicId: true,
        createdAt: true,
        updatedAt: true,
        isAdmin: true,
        permissions: true,
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
