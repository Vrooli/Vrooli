import { rel } from "../utils";
export const roleTranslation = {
    __typename: "RoleTranslation",
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
};
export const role = {
    __typename: "Role",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        permissions: true,
        membersCount: true,
        organization: async () => rel((await import("./organization")).organization, "nav", { omit: "roles" }),
        translations: () => rel(roleTranslation, "full"),
    },
    full: {
        members: async () => rel((await import("./member")).member, "nav", { omit: "reminder" }),
    },
    list: {},
};
//# sourceMappingURL=role.js.map