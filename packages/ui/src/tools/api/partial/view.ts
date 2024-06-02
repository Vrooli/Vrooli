import { View } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const view: GqlPartial<View> = {
    __typename: "View",
    list: {
        __define: {
            0: async () => rel((await import("./api")).api, "list"),
            1: async () => rel((await import("./issue")).issue, "list"),
            2: async () => rel((await import("./note")).note, "list"),
            3: async () => rel((await import("./team")).team, "list"),
            4: async () => rel((await import("./post")).post, "list"),
            5: async () => rel((await import("./project")).project, "list"),
            6: async () => rel((await import("./question")).question, "list"),
            7: async () => rel((await import("./routine")).routine, "list"),
            8: async () => rel((await import("./code")).code, "list"),
            9: async () => rel((await import("./standard")).standard, "list"),
            10: async () => rel((await import("./user")).user, "list"),
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                Code: 8,
                Issue: 1,
                Note: 2,
                Post: 4,
                Project: 5,
                Question: 6,
                Routine: 7,
                Standard: 9,
                Team: 3,
                User: 10,
            },
        },
    },
};
