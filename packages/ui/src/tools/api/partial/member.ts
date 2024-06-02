import { Member, MemberYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const memberYou: GqlPartial<MemberYou> = {
    __typename: "MemberYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const member: GqlPartial<Member> = {
    __typename: "Member",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isAdmin: true,
        permissions: true,
        roles: async () => rel((await import("./role")).role, "list"),
        you: () => rel(memberYou, "full"),
    },
    full: {
        team: async () => rel((await import("./team")).team, "full"),
        user: async () => rel((await import("./user")).user, "full"),
    },
    list: {
        team: async () => rel((await import("./team")).team, "list"),
        user: async () => rel((await import("./user")).user, "list"),
    },
};
