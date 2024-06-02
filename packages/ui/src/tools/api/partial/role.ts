import { Role, RoleTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const roleTranslation: GqlPartial<RoleTranslation> = {
    __typename: "RoleTranslation",
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
};

export const role: GqlPartial<Role> = {
    __typename: "Role",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        permissions: true,
        membersCount: true,
        team: async () => rel((await import("./team")).team, "nav", { omit: "roles" }),
        translations: () => rel(roleTranslation, "full"),
    },
    full: {
        members: async () => rel((await import("./member")).member, "nav", { omit: "reminder" }),
    },
    list: {},
};
