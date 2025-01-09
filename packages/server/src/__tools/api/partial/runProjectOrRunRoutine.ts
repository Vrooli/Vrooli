import { RunProjectOrRunRoutine } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const runProjectOrRunRoutine: ApiPartial<RunProjectOrRunRoutine> = {
    full: {
        __union: {
            RunProject: async () => rel((await import("./runProject")).runProject, "full"),
            RunRoutine: async () => rel((await import("./runRoutine")).runRoutine, "full"),
        },
    },
    list: {
        __union: {
            RunProject: async () => rel((await import("./runProject")).runProject, "list"),
            RunRoutine: async () => rel((await import("./runRoutine")).runRoutine, "list"),
        },
    },
};
