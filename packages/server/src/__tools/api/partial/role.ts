import { Role, RoleTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const roleTranslation: ApiPartial<RoleTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
    },
};

export const role: ApiPartial<Role> = {
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
        members: async () => rel((await import("./member")).member, "list", { omit: "team" }),
    },
};
