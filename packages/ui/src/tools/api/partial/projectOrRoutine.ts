import { ProjectOrRoutine } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const projectOrRoutine: GqlPartial<ProjectOrRoutine> = {
    __typename: "ProjectOrRoutine" as any,
    full: {
        __define: {
            0: async () => rel((await import("./project")).project, "full"),
            1: async () => rel((await import("./routine")).routine, "full"),
        },
        __union: {
            Project: 0,
            Routine: 1,
        },
    },
    list: {
        __define: {
            0: async () => rel((await import("./project")).project, "list"),
            1: async () => rel((await import("./routine")).routine, "list"),
        },
        __union: {
            Project: 0,
            Routine: 1,
        },
    },
};
