import { ProjectOrRoutine } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const projectOrRoutine: GqlPartial<ProjectOrRoutine> = {
    full: {
        __union: {
            Project: async () => rel((await import("./project")).project, "full"),
            Routine: async () => rel((await import("./routine")).routine, "full"),
        },
    },
    list: {
        __union: {
            Project: async () => rel((await import("./project")).project, "list"),
            Routine: async () => rel((await import("./routine")).routine, "list"),
        },
    },
};
