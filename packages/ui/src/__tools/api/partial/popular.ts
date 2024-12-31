import { Popular } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const popular: GqlPartial<Popular> = {
    __typename: "Popular" as Popular["__typename"],
    list: {
        __define: {
            0: async () => rel((await import("./api")).api, "list"),
            1: async () => rel((await import("./note")).note, "list"),
            2: async () => rel((await import("./team")).team, "list"),
            3: async () => rel((await import("./project")).project, "list"),
            4: async () => rel((await import("./question")).question, "list"),
            5: async () => rel((await import("./routine")).routine, "list"),
            6: async () => rel((await import("./code")).code, "list"),
            7: async () => rel((await import("./standard")).standard, "list"),
            8: async () => rel((await import("./user")).user, "list"),
        },
        __union: {
            Api: 0,
            Code: 6,
            Note: 1,
            Project: 3,
            Question: 4,
            Routine: 5,
            Standard: 7,
            Team: 2,
            User: 8,
        },
    },
};
