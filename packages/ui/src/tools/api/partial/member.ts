import { Member } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const member: GqlPartial<Member> = {
    __typename: "Member",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isAdmin: true,
        permissions: true,
        roles: async () => rel((await import("./role")).role, "list"),
    },
    full: {
        organization: async () => rel((await import("./organization")).organization, "full"),
        user: async () => rel((await import("./user")).user, "full"),
    },
    list: {
        organization: async () => rel((await import("./organization")).organization, "list"),
        user: async () => rel((await import("./user")).user, "list"),
    },
};
