import { ProjectOrTeam } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const projectOrTeam: ApiPartial<ProjectOrTeam> = {
    full: {
        __union: {
            Project: async () => rel((await import("./project.js")).project, "full"),
            Team: async () => rel((await import("./team.js")).team, "full"),
        },
    },
    list: {
        __union: {
            Project: async () => rel((await import("./project.js")).project, "list"),
            Team: async () => rel((await import("./team.js")).team, "list"),
        },
    },
};
