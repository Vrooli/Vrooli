import { HomeResult } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const homeResult: GqlPartial<HomeResult> = {
    __typename: "HomeResult",
    list: {
        __define: {
            1: async () => rel((await import("./resource")).resource, "list"),
            2: async () => rel((await import("./reminder")).reminder, "full"),
            3: async () => rel((await import("./schedule")).schedule, "list"),
        },
        recommended: { __use: 1 },
        reminders: { __use: 2 },
        resources: { __use: 1 },
        schedules: { __use: 3 },
    },
};
