import { ProjectOrTeam } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const projectOrTeam: GqlPartial<ProjectOrTeam> = {
    __typename: "ProjectOrTeam" as ProjectOrTeam["__typename"],
    full: {
        __define: {
            0: async () => rel((await import("./project")).project, "full"),
            1: async () => rel((await import("./team")).team, "full"),
        },
        __union: {
            Project: 0,
            Team: 1,
        },
    },
    list: {
        __define: {
            0: async () => rel((await import("./project")).project, "list"),
            1: async () => rel((await import("./team")).team, "list"),
        },
        __union: {
            Project: 0,
            Team: 1,
        },
    },
};
