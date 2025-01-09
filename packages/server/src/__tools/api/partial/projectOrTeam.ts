import { ProjectOrTeam } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const projectOrTeam: ApiPartial<ProjectOrTeam> = {
    full: {
        __union: {
            Project: async () => rel((await import("./project")).project, "full"),
            Team: async () => rel((await import("./team")).team, "full"),
        },
    },
    list: {
        __union: {
            Project: async () => rel((await import("./project")).project, "list"),
            Team: async () => rel((await import("./team")).team, "list"),
        },
    },
};
