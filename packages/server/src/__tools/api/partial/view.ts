import { type View } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const view: ApiPartial<View> = {
    list: {
        to: {
            __union: {
                Issue: async () => rel((await import("./issue.js")).issue, "list"),
                Resource: async () => rel((await import("./resource.js")).resource, "list"),
                Team: async () => rel((await import("./team.js")).team, "list"),
                User: async () => rel((await import("./user.js")).user, "list"),
            },
        },
    },
};
