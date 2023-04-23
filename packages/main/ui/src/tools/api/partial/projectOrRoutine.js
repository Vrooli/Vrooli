import { rel } from "../utils";
export const projectOrRoutine = {
    __typename: "ProjectOrRoutine",
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
//# sourceMappingURL=projectOrRoutine.js.map