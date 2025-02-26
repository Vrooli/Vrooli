import { RunProjectOrRunRoutine } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runProjectOrRunRoutine: ApiPartial<RunProjectOrRunRoutine> = {
    full: {
        __union: {
            RunProject: async () => rel((await import("./runProject.js")).runProject, "full"),
            RunRoutine: async () => rel((await import("./runRoutine.js")).runRoutine, "full"),
        },
    },
    list: {
        __union: {
            RunProject: async () => rel((await import("./runProject.js")).runProject, "list"),
            RunRoutine: async () => rel((await import("./runRoutine.js")).runRoutine, "list"),
        },
    },
};
