import { HomeResult } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const homeResult: GqlPartial<HomeResult> = {
    __typename: "HomeResult",
    list: {
        __define: {
            1: async () => rel((await import("./note")).note, "list"),
            2: async () => rel((await import("./reminder")).reminder, "list"),
            3: async () => rel((await import("./resource")).resource, "list"),
            4: async () => rel((await import("./schedule")).schedule, "list"),
        },
        notes: { __use: 1 },
        reminders: { __use: 2 },
        resources: { __use: 3 },
        schedules: { __use: 4 },
    },
};
