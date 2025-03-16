import { ProjectOrRoutine } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const projectOrRoutine: ApiPartial<ProjectOrRoutine> = {
    full: {
        __union: {
            Project: async () => rel((await import("./project.js")).project, "full"),
            Routine: async () => rel((await import("./routine.js")).routine, "full"),
        },
    },
    list: {
        __union: {
            Project: async () => rel((await import("./project.js")).project, "list"),
            Routine: async () => rel((await import("./routine.js")).routine, "list"),
        },
    },
};
